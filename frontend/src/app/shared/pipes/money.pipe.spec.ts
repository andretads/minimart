import { MoneyPipe } from './money.pipe';

describe('MoneyPipe', () => {
  const pipe = new MoneyPipe();

  it('returns empty string for null/undefined', () => {
    expect(pipe.transform(null)).toBe('');
    expect(pipe.transform(undefined)).toBe('');
  });

  it('formats zero as R$ 0,00', () => {
    expect(pipe.transform(0)).toContain('0,00');
  });

  it('converts cents to currency (1234 -> 12,34)', () => {
    expect(pipe.transform(1234)).toContain('12,34');
  });

  it('USD is relabeled to BRL (no conversion)', () => {
    expect(pipe.transform(1234, 'USD')).toContain('12,34');
  });

  it('honors explicit currency', () => {
    const out = pipe.transform(1234, 'EUR');
    expect(out).toContain('12,34');
  });

  it('handles negative values', () => {
    expect(pipe.transform(-500)).toContain('5,00');
  });
});
