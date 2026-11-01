//go:build integration

package test

import (
	"minimart/internal/cart/application"
	"minimart/internal/cart/domain"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateItemQuantityIntegration(t *testing.T) {
	t.Run("Blank user_id is rejected", func(t *testing.T) {
		env := newCartEnv(t)

		_, err := env.svc.UpdateItemQuantity(t.Context(), application.UpdateItemQuantityRequest{
			UserID:    "",
			ProductID: featureProductID,
			Quantity:  3,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrUserIdRequired)
	})

	t.Run("Quantity less than 1 is rejected", func(t *testing.T) {
		env := newCartEnv(t)
		user := uniqueUser()
		env.seedCart(t, user, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		_, err := env.svc.UpdateItemQuantity(t.Context(), application.UpdateItemQuantityRequest{
			UserID:    user,
			ProductID: featureProductID,
			Quantity:  0,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidQuantity)
	})

	t.Run("Unknown product in cart returns item not found", func(t *testing.T) {
		env := newCartEnv(t)
		user := uniqueUser()
		env.seedCart(t, user, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		_, err := env.svc.UpdateItemQuantity(t.Context(), application.UpdateItemQuantityRequest{
			UserID:    user,
			ProductID: "nope",
			Quantity:  1,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCartItemNotFound)
	})

	t.Run("Updates quantity and bumps version", func(t *testing.T) {
		env := newCartEnv(t)
		user := uniqueUser()
		env.seedCart(t, user, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		_, err := env.svc.UpdateItemQuantity(t.Context(), application.UpdateItemQuantityRequest{
			UserID:    user,
			ProductID: featureProductID,
			Quantity:  5,
		})
		require.NoError(t, err)

		stored := env.storedCart(t, user)
		requireItem(t, stored.Items, featureProductID, featureProductName, 5, featureProductPriceCents)
		assert.Equal(t, 2, stored.Version)
	})

	t.Run("Update locks against the current cart version", func(t *testing.T) {
		env := newCartEnv(t)
		user := uniqueUser()
		env.seedCart(t, user, 2, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		_, err := env.svc.UpdateItemQuantity(t.Context(), application.UpdateItemQuantityRequest{
			UserID:    user,
			ProductID: featureProductID,
			Quantity:  3,
		})
		require.NoError(t, err)

		// The use case reads the current version (2) and CASes on it, so the
		// stored cart moves to version 3.
		stored := env.storedCart(t, user)
		requireItem(t, stored.Items, featureProductID, featureProductName, 3, featureProductPriceCents)
		assert.Equal(t, 3, stored.Version)

		// A write that CASes on the stale version is rejected.
		stale := stored
		stale.Items = append(stale.Items, featureItem(featureExtraProductID, featureExtraProductName, featureExtraProductPrice, 1))
		require.ErrorIs(t, env.repo.Update(t.Context(), stale, 2), domain.ErrCartOptimisticLock)

		// Concurrent writers that all CAS on the same version: exactly one wins.
		raceCart := env.seedCart(t, uniqueUser(), 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 1))
		const writers = 10
		products := []string{"prod-a", "prod-b", "prod-c", "prod-d", "prod-e", "prod-f", "prod-g", "prod-h", "prod-i", "prod-j"}
		errs := make(chan error, writers)
		var wg sync.WaitGroup
		for i := range writers {
			wg.Go(func() {
				updated := raceCart
				updated.Items = append(updated.Items, featureItem(products[i], "Race Product", 100, 1))
				errs <- env.repo.Update(t.Context(), updated, raceCart.Version)
			})
		}
		wg.Wait()
		close(errs)

		successes := 0
		for err := range errs {
			if err == nil {
				successes++
				continue
			}
			require.ErrorIs(t, err, domain.ErrCartOptimisticLock)
		}
		assert.Equal(t, 1, successes)
	})
}
