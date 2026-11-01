export interface Customer {
  id: string;
  userId: string;
  displayName: string;
  status: 'active' | 'deactivated';
  version: number;
  createdAt: string;
  updatedAt: string;
  deactivatedAt?: string;
}

export interface UpdateCustomerRequest {
  displayName: string;
  version: number;
}

export interface GetCustomerByUserIdResponse {
  id: string;
  userId: string;
  displayName: string;
  status: string;
  version: number;
  createdAt: string;
  updatedAt: string;
  deactivatedAt?: string;
}
