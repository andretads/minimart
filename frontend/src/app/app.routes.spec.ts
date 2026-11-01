import { routes } from './app.routes';
import { authGuard } from './core/auth/auth.guard';
import { guestGuard } from './core/auth/guest.guard';

describe('app routes', () => {
  const find = (path: string) => routes.find((r) => r.path === path);

  it('exposes a wildcard redirect to the home page', () => {
    const wildcard = find('**');
    expect(wildcard?.redirectTo).toBe('');
  });

  it('redirects /allorders to orders', () => {
    const route = find('allorders');
    expect(route?.redirectTo).toBe('orders');
    expect(route?.pathMatch).toBe('full');
  });

  it('guards private routes with authGuard', () => {
    for (const path of [
      'cart',
      'orders',
      'orders/:id',
      'payments/:id',
      'profile',
      'notifications',
      'wishlist',
      'settings',
      'cash-order',
    ]) {
      expect(find(path)?.canActivate).toContain(authGuard);
    }
  });

  it('guards login and register with guestGuard', () => {
    expect(find('login')?.canActivate).toContain(guestGuard);
    expect(find('register')?.canActivate).toContain(guestGuard);
  });

  it('leaves public routes unguarded', () => {
    for (const path of ['', 'products/:id', 'home', 'products', 'categories', 'brands']) {
      expect(find(path)?.canActivate).toBeUndefined();
    }
  });

  it('lazy loads every route component', () => {
    for (const route of routes) {
      if (route.redirectTo !== undefined) continue;
      expect(typeof route.loadComponent).toBe('function');
    }
  });

  it('provides titles for navigable routes', () => {
    for (const route of routes) {
      if (route.redirectTo !== undefined) continue;
      expect(route.title).toMatch(/MiniMart/);
    }
  });

  it('resolves the home loader to HomeComponent', async () => {
    const mod = await find('')!.loadComponent!();
    expect((mod as { name?: string }).name).toContain('HomeComponent');
  });

  it('resolves the product detail loader to ProductDetailComponent', async () => {
    const mod = await find('products/:id')!.loadComponent!();
    expect((mod as { name?: string }).name).toContain('ProductDetailComponent');
  });

  it('resolves the login loader to LoginComponent', async () => {
    const mod = await find('login')!.loadComponent!();
    expect((mod as { name?: string }).name).toContain('LoginComponent');
  });

  it('resolves the wishlist loader to WishlistComponent', async () => {
    const mod = await find('wishlist')!.loadComponent!();
    expect((mod as { name?: string }).name).toContain('WishlistComponent');
  });
});
