import { Routes } from '@angular/router';
import { authGuard } from './core/auth/auth.guard';
import { guestGuard } from './core/auth/guest.guard';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () =>
      import('./features/catalog/home/home.component').then((m) => m.HomeComponent),
    title: 'Início | MiniMart',
  },
  {
    path: 'products/:id',
    loadComponent: () =>
      import('./features/catalog/product-detail/product-detail.component').then(
        (m) => m.ProductDetailComponent,
      ),
    title: 'Detalhes do produto | MiniMart',
  },
  {
    path: 'login',
    loadComponent: () =>
      import('./features/auth/login/login.component').then((m) => m.LoginComponent),
    canActivate: [guestGuard],
    title: 'Entrar | MiniMart',
  },
  {
    path: 'register',
    loadComponent: () =>
      import('./features/auth/register/register.component').then((m) => m.RegisterComponent),
    canActivate: [guestGuard],
    title: 'Criar conta | MiniMart',
  },
  {
    path: 'cart',
    loadComponent: () => import('./features/cart/cart.component').then((m) => m.CartComponent),
    canActivate: [authGuard],
    title: 'Carrinho | MiniMart',
  },
  {
    path: 'orders',
    loadComponent: () =>
      import('./features/orders/order-list/order-list.component').then((m) => m.OrderListComponent),
    canActivate: [authGuard],
    title: 'Pedidos | MiniMart',
  },
  {
    path: 'orders/:id',
    loadComponent: () =>
      import('./features/orders/order-details/order-details.component').then(
        (m) => m.OrderDetailsComponent,
      ),
    canActivate: [authGuard],
    title: 'Detalhes do pedido | MiniMart',
  },
  {
    path: 'payments/:id',
    loadComponent: () =>
      import('./features/payments/payment-status/payment-status.component').then(
        (m) => m.PaymentStatusComponent,
      ),
    canActivate: [authGuard],
    title: 'Status do pagamento | MiniMart',
  },
  {
    path: 'profile',
    loadComponent: () =>
      import('./features/customer/profile/profile.component').then((m) => m.ProfileComponent),
    canActivate: [authGuard],
    title: 'Minha conta | MiniMart',
  },
  {
    path: 'notifications',
    loadComponent: () =>
      import('./features/notifications/notifications.component').then(
        (m) => m.NotificationsComponent,
      ),
    canActivate: [authGuard],
    title: 'Notificações | MiniMart',
  },
  {
    path: 'home',
    loadComponent: () =>
      import('./features/catalog/home/home.component').then((m) => m.HomeComponent),
    title: 'Início | MiniMart',
  },
  {
    path: 'products',
    loadComponent: () =>
      import('./features/catalog/home/home.component').then((m) => m.HomeComponent),
    title: 'Produtos | MiniMart',
  },
  {
    path: 'categories',
    loadComponent: () =>
      import('./features/catalog/home/home.component').then((m) => m.HomeComponent),
    title: 'Categorias | MiniMart',
  },
  {
    path: 'wishlist',
    loadComponent: () =>
      import('./features/wishlist/wishlist.component').then((m) => m.WishlistComponent),
    canActivate: [authGuard],
    title: 'Lista de desejos | MiniMart',
  },
  {
    path: 'allorders',
    redirectTo: 'orders',
    pathMatch: 'full',
  },
  {
    path: 'settings',
    loadComponent: () =>
      import('./features/customer/profile/profile.component').then((m) => m.ProfileComponent),
    canActivate: [authGuard],
    title: 'Configurações | MiniMart',
  },
  {
    path: 'brands',
    loadComponent: () =>
      import('./features/catalog/home/home.component').then((m) => m.HomeComponent),
    title: 'Marcas | MiniMart',
  },
  {
    path: 'cash-order',
    loadComponent: () => import('./features/cart/cart.component').then((m) => m.CartComponent),
    canActivate: [authGuard],
    title: 'Finalizar compra | MiniMart',
  },
  {
    path: '**',
    redirectTo: '',
  },
];
