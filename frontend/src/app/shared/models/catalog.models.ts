export interface ProductDetail {
  name: string;
  value: string;
}

export interface ProductImage {
  id: string;
  productId: string;
  url: string;
  contentType: string;
  sizeBytes: number;
  position: number;
}

export interface ProductReview {
  id: string;
  productId: string;
  customerId: string;
  description: string;
  rating: number;
  status: string;
  createdAt: string;
}

export interface Product {
  id: string;
  name: string;
  priceCents: number;
  categoryId?: string;
  stock?: number;
  sku?: string;
  description?: string;
  details?: ProductDetail[];
  totalRating?: number;
  totalReviews?: number;
  imageUrl?: string;
  images?: ProductImage[];
}

export interface ProductListResponse {
  data: Product[];
  meta: {
    pageSize: number;
    nextCursor: string | null;
    hasMore: boolean;
  };
}

export interface Category {
  id: string;
  name: string;
  icon: string;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface ProductReviewListResponse {
  data: ProductReview[];
  meta: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
}
