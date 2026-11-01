import { Component, input, output, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ButtonComponent } from '../button/button.component';

@Component({
  selector: 'app-quantity-stepper',
  standalone: true,
  imports: [CommonModule, ButtonComponent],
  template: `
    <div class="stepper" [class.disabled]="disabled()">
      <app-button
        variant="ghost"
        size="sm"
        [disabled]="disabled() || value() <= min()"
        (clicked)="decrement()"
        aria-label="Diminuir quantidade"
        label="−"
      />
      <input
        type="number"
        [value]="value()"
        [min]="min()"
        [max]="max()"
        [disabled]="disabled()"
        (input)="onInput($event)"
        (blur)="onBlur()"
        class="stepper-input"
        aria-label="Quantidade"
      />
      <app-button
        variant="ghost"
        size="sm"
        [disabled]="disabled() || value() >= max()"
        (clicked)="increment()"
        aria-label="Aumentar quantidade"
        label="+"
      />
    </div>
  `,
  styles: [
    `
      .stepper {
        display: inline-flex;
        align-items: center;
        gap: 0.15rem;
        padding: 0.2rem;
        border: 1px solid var(--border);
        border-radius: var(--radius-full);
        background: var(--card);
      }
      .stepper.disabled {
        opacity: 0.5;
      }
      .stepper-input {
        width: 2.75rem;
        height: 2rem;
        padding: 0;
        text-align: center;
        border: none;
        background: transparent;
        font: inherit;
        font-weight: 600;
        font-size: 0.875rem;
        color: var(--foreground);
        -moz-appearance: textfield;
      }
      .stepper-input::-webkit-outer-spin-button,
      .stepper-input::-webkit-inner-spin-button {
        -webkit-appearance: none;
        margin: 0;
      }
      .stepper-input:focus {
        outline: none;
      }
    `,
  ],
})
export class QuantityStepperComponent implements OnInit {
  value = input.required<number>();
  min = input(1);
  max = input(99);
  disabled = input(false);
  valueChange = output<number>();

  private lastValidValue = signal<number>(0);

  ngOnInit(): void {
    this.lastValidValue.set(this.value());
  }

  increment(): void {
    if (this.value() < this.max() && !this.disabled()) {
      this.valueChange.emit(this.value() + 1);
    }
  }

  decrement(): void {
    if (this.value() > this.min() && !this.disabled()) {
      this.valueChange.emit(this.value() - 1);
    }
  }

  onInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    const val = parseInt(input.value, 10);
    if (!isNaN(val) && val >= this.min() && val <= this.max()) {
      this.lastValidValue.set(val);
      this.valueChange.emit(val);
    }
  }

  onBlur(): void {
    this.valueChange.emit(this.lastValidValue());
  }
}
