export interface User {
  id: string;
  username: string;
  email: string;
  createdAt: string;
}

export interface LoginRequest {
  username: string;
  password: string;
  rememberMe?: boolean;
}

export interface SignupRequest {
  username: string;
  email: string;
  password: string;
}

export interface SignupResponse {
  id: string;
  username: string;
  email: string;
  createdAt: string;
}

export interface ChangePasswordRequest {
  currentPassword: string;
  newPassword: string;
  confirmNewPassword: string;
}

export interface ChangePasswordResponse {
  status: string;
}

export interface SessionResponse extends User {
  accessTokenExpiresAt: string;
  expiresIn: number;
}

export interface RefreshResponse {
  status: string;
  accessTokenExpiresAt: string;
  expiresIn: number;
}

export interface MeResponse extends User {
  accessTokenExpiresAt: string;
  expiresIn: number;
}
