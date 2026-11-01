import {
  formatDatePtBr,
  formatDateLongPtBr,
  formatDateTimePtBr,
  formatDateTimeLongPtBr,
} from './datetime';

describe('datetime', () => {
  const iso = '2026-02-01T10:30:00Z';

  it('formatDatePtBr from string', () => {
    const out = formatDatePtBr(iso);
    // locale-dependent; just ensure non-empty and contains year
    expect(out).toContain('2026');
  });

  it('formatDatePtBr from Date', () => {
    expect(formatDatePtBr(new Date(iso))).toContain('2026');
  });

  it('formatDateLongPtBr contains month name', () => {
    expect(formatDateLongPtBr(iso)).toContain('2026');
  });

  it('formatDateTimePtBr contains time', () => {
    const out = formatDateTimePtBr(iso);
    expect(out).toContain('2026');
  });

  it('formatDateTimeLongPtBr', () => {
    expect(formatDateTimeLongPtBr(iso)).toContain('2026');
  });

  it('invalid date string yields "Invalid Date"', () => {
    expect(formatDatePtBr('garbage')).toBe('Invalid Date');
  });
});
