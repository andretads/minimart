package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateIDForType(t *testing.T) {
	assert.Equal(t, "order_confirmation_v1", TemplateIDForType(TypeOrderConfirmation))
	assert.Equal(t, "order_cancelled_v1", TemplateIDForType(TypeOrderCancellation))
	assert.Equal(t, "payment_succeeded_v1", TemplateIDForType(TypePaymentSucceeded))
	assert.Equal(t, "payment_failed_v1", TemplateIDForType(TypePaymentFailed))
	assert.Equal(t, "shipment_created_v1", TemplateIDForType(TypeShipmentCreated))
	assert.Equal(t, "shipment_shipped_v1", TemplateIDForType(TypeShipmentShipped))
	assert.Equal(t, "shipment_delivered_v1", TemplateIDForType(TypeShipmentDelivered))
	assert.Empty(t, TemplateIDForType(Type("unknown")))
}

func TestPriorityForType(t *testing.T) {
	assert.Equal(t, PriorityHigh, PriorityForType(TypePaymentFailed))
	assert.Equal(t, PriorityNormal, PriorityForType(TypeOrderConfirmation))
}

func TestCategoryForType(t *testing.T) {
	assert.Equal(t, CategoryTransactional, CategoryForType(TypeOrderConfirmation))
}

func TestRenderTemplate(t *testing.T) {
	subject := "Pedido o-1 confirmado"
	body := "Obrigado pela sua compra!\nO pedido o-1 no valor de R$ 12,50 foi confirmado."

	t.Run("email renders html", func(t *testing.T) {
		rendered, err := RenderTemplate("order_confirmation_v1", ChannelEmail, subject, body)
		require.NoError(t, err)
		assert.Equal(t, subject, rendered.Subject)
		assert.Equal(t, body, rendered.Text)
		assert.Contains(t, rendered.HTML, "<br>")
	})

	t.Run("sms truncates long text", func(t *testing.T) {
		long := strings.Repeat("a", 200)
		rendered, err := RenderTemplate("order_confirmation_v1", ChannelSMS, subject, long)
		require.NoError(t, err)
		assert.Len(t, rendered.Text, 160)
	})

	t.Run("push renders title and body", func(t *testing.T) {
		rendered, err := RenderTemplate("order_confirmation_v1", ChannelPush, subject, body)
		require.NoError(t, err)
		assert.Equal(t, subject, rendered.Title)
		assert.Equal(t, body, rendered.Body)
	})

	t.Run("unknown template returns error", func(t *testing.T) {
		_, err := RenderTemplate("missing_v1", ChannelEmail, subject, body)
		assert.ErrorIs(t, err, ErrTemplateNotFound)
	})

	t.Run("invalid channel returns error", func(t *testing.T) {
		_, err := RenderTemplate("order_confirmation_v1", Channel("fax"), subject, body)
		assert.ErrorIs(t, err, ErrInvalidChannel)
	})

	t.Run("empty template id returns error", func(t *testing.T) {
		_, err := RenderTemplate("", ChannelEmail, subject, body)
		assert.ErrorIs(t, err, ErrTemplateNotFound)
	})
}
