export const APP_SETTINGS = {
  currency: 'BRL',
  paymentMethod: 'card',
  pageSize: 20,
  maxPageSize: 100,
  defaultPage: 1,
} as const;

export type AppSettings = typeof APP_SETTINGS;
