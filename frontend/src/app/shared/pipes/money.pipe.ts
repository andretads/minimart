import { Pipe, PipeTransform } from '@angular/core';
import { APP_SETTINGS } from '../../core/config/app-settings';

@Pipe({
  name: 'money',
  standalone: true,
})
export class MoneyPipe implements PipeTransform {
  transform(cents: number | null | undefined, currency?: string): string {
    if (cents === null || cents === undefined) return '';
    const currencyCode = currency ?? APP_SETTINGS.currency;
    // Normalize legacy USD to BRL without conversion (same cents)
    const normalized = currencyCode === 'USD' ? 'BRL' : currencyCode;
    return new Intl.NumberFormat('pt-BR', {
      style: 'currency',
      currency: normalized,
      minimumFractionDigits: 2,
    }).format(cents / 100);
  }
}
