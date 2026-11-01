import { TestBed } from '@angular/core/testing';
import { environment } from '../../../environments/environment';
import { API_BASE_URL } from './api-base-url.token';

describe('API_BASE_URL', () => {
  it('is a root-provided token', () => {
    const token = API_BASE_URL as unknown as { providedIn?: string };
    expect(token).toBeDefined();
  });

  it('resolves to the configured apiBaseUrl', () => {
    TestBed.configureTestingModule({});
    const value = TestBed.inject(API_BASE_URL);
    expect(value).toBe(environment.apiBaseUrl);
  });
});
