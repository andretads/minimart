import { APP_SETTINGS } from './app-settings';

describe('APP_SETTINGS', () => {
  it('has the expected values', () => {
    expect(APP_SETTINGS.currency).toBe('BRL');
    expect(APP_SETTINGS.paymentMethod).toBe('card');
    expect(APP_SETTINGS.pageSize).toBe(20);
    expect(APP_SETTINGS.maxPageSize).toBe(100);
    expect(APP_SETTINGS.defaultPage).toBe(1);
  });
});
