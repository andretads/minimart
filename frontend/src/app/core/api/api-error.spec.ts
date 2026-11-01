import {
  ApiEnvelope,
  ApiError,
  createApiError,
  isApiError,
  isAuthenticationError,
} from './api-error';

describe('api-error', () => {
  describe('createApiError', () => {
    it('uses the envelope message when present', () => {
      const envelope: ApiEnvelope<unknown> = {
        time: '2026-01-01T00:00:00Z',
        success: false,
        error: { code: 'E_BAD', message: 'bad request', fields: { name: ['required'] } },
      };
      const err = createApiError(400, envelope, { original: true });
      expect(err.status).toBe(400);
      expect(err.code).toBe('E_BAD');
      expect(err.message).toBe('bad request');
      expect(err.fields).toEqual({ name: ['required'] });
      expect(err.originalError).toEqual({ original: true });
      expect(err instanceof Error).toBe(true);
    });

    it('falls back to "HTTP <status>" when no envelope message', () => {
      expect(createApiError(500).message).toBe('HTTP 500');
      expect(createApiError(500).status).toBe(500);
      expect(createApiError(500).code).toBeUndefined();
      expect(createApiError(500).fields).toBeUndefined();
      expect(createApiError(500).originalError).toBeUndefined();
    });

    it('falls back to "HTTP <status>" when envelope has no error', () => {
      const envelope: ApiEnvelope<unknown> = { time: 't', success: false };
      expect(createApiError(404, envelope).message).toBe('HTTP 404');
    });
  });

  describe('isApiError', () => {
    it('returns true for an Error carrying a status', () => {
      const err = Object.assign(new Error('x'), { status: 500 }) as ApiError;
      expect(isApiError(err)).toBe(true);
    });

    it('returns false for null, undefined, plain objects, and bare Errors', () => {
      expect(isApiError(null)).toBe(false);
      expect(isApiError(undefined)).toBe(false);
      expect(isApiError({})).toBe(false);
      expect(isApiError(new Error('x'))).toBe(false);
      expect(isApiError(42)).toBe(false);
      expect(isApiError('str')).toBe(false);
    });
  });

  describe('isAuthenticationError', () => {
    it('returns true for 401 and 403 ApiErrors', () => {
      const a = Object.assign(new Error('a'), { status: 401 });
      const b = Object.assign(new Error('b'), { status: 403 });
      expect(isAuthenticationError(a)).toBe(true);
      expect(isAuthenticationError(b)).toBe(true);
    });

    it('returns false for other statuses and non-ApiErrors', () => {
      const c = Object.assign(new Error('c'), { status: 404 });
      expect(isAuthenticationError(c)).toBe(false);
      expect(isAuthenticationError(null)).toBe(false);
      expect(isAuthenticationError(new Error('x'))).toBe(false);
    });
  });
});
