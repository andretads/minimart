import { FooterComponent } from './footer.component';

describe('FooterComponent', () => {
  it('currentYear is current year', () => {
    const c = new FooterComponent();
    expect(c.currentYear).toBe(new Date().getFullYear());
  });
});
