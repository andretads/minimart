export interface CartItem {
  productId: string;
  productNameSnapshot: string;
  unitPriceSnapshotCents: number;
  quantity: number;
  lineTotalCents: number;
}

export interface Cart {
  id: string;
  customerId: string;
  status: 'active' | 'checkout_pending' | 'consumed';
  items: CartItem[];
  estimatedTotalCents: number;
  createdAt: string;
  updatedAt: string;
}

export interface AddItemRequest {
  productId: string;
  quantity: number;
}

export interface UpdateItemRequest {
  quantity: number;
}

export interface CheckoutResponse {
  orderId: string;
  userId: string;
  customerId: string;
  status: string;
  totalCents: number;
  createdAt: string;
}
