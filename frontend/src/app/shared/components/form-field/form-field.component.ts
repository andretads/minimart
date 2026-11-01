import {
  Component,
  input,
  forwardRef,
  signal,
  computed,
  Injector,
  inject,
  OnInit,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  ControlValueAccessor,
  NG_VALUE_ACCESSOR,
  ReactiveFormsModule,
  NgControl,
  FormGroup,
  AbstractControl,
} from '@angular/forms';

@Component({
  selector: 'app-form-field',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => FormFieldComponent),
      multi: true,
    },
  ],
  template: `
    <div class="form-field">
      @if (label()) {
        <label [for]="id()" class="field-label">
          {{ label() }}
          @if (required()) {
            <span class="required" aria-hidden="true">*</span>
          }
        </label>
      }
      <div class="field-input-wrapper">
        <input
          [id]="id()"
          [type]="type()"
          [placeholder]="placeholder()"
          [disabled]="isDisabled()"
          [value]="value()"
          [attr.aria-invalid]="computedError() ? 'true' : 'false'"
          [attr.aria-describedby]="computedError() ? errorId() : null"
          (input)="onInput($event)"
          (blur)="onBlur()"
          class="field-input"
        />
        @if (showPassword()) {
          <button
            type="button"
            class="password-toggle"
            (click)="togglePassword()"
            [attr.aria-label]="showPassword() ? 'Ocultar senha' : 'Mostrar senha'"
            [attr.aria-pressed]="showPassword()"
          >
            <span
              class="icon icon-sm"
              [class.icon-eye]="!showPassword()"
              [class.icon-eye-off]="showPassword()"
              aria-hidden="true"
            ></span>
          </button>
        }
      </div>
      @if (computedError()) {
        <div [id]="errorId()" class="field-error" role="alert">{{ computedError() }}</div>
      }
      @if (hint() && !computedError()) {
        <div class="field-hint">{{ hint() }}</div>
      }
    </div>
  `,
  styles: [
    `
      .form-field {
        display: flex;
        flex-direction: column;
        gap: 0.4rem;
      }
      .field-label {
        font-size: 0.8125rem;
        font-weight: 600;
        color: var(--foreground);
      }
      .required {
        color: var(--destructive);
        margin-left: 0.25rem;
      }
      .field-input-wrapper {
        position: relative;
        display: flex;
        align-items: center;
      }
      .field-input {
        width: 100%;
        min-height: 2.75rem;
        padding: 0.65rem 0.9rem;
        font: inherit;
        font-size: 0.9375rem;
        color: var(--foreground);
        background: color-mix(in oklab, var(--background) 65%, var(--card));
        border: 1px solid var(--border);
        border-radius: var(--radius-md);
        transition:
          border-color var(--transition-fast),
          box-shadow var(--transition-fast),
          background-color var(--transition-fast);
      }
      .field-input::placeholder {
        color: var(--muted-foreground);
        opacity: 0.8;
      }
      .field-input:hover:not(:disabled) {
        border-color: color-mix(in oklab, var(--muted-foreground) 35%, var(--border));
      }
      .field-input:focus {
        outline: none;
        border-color: var(--accent);
        box-shadow: 0 0 0 3px var(--color-primary-bg);
        background: var(--background);
      }
      .field-input:disabled {
        background: var(--muted);
        color: var(--muted-foreground);
        cursor: not-allowed;
      }
      .field-input[aria-invalid='true'] {
        border-color: var(--destructive);
      }
      .field-input[aria-invalid='true']:focus {
        box-shadow: 0 0 0 3px var(--color-danger-bg);
      }
      .password-toggle {
        position: absolute;
        right: 0.55rem;
        display: grid;
        place-items: center;
        width: 1.85rem;
        height: 1.85rem;
        background: transparent;
        border: 0;
        border-radius: var(--radius-sm);
        color: var(--muted-foreground);
        cursor: pointer;
      }
      .password-toggle:hover {
        background: var(--muted);
        color: var(--foreground);
      }
      .field-error {
        font-size: 0.78rem;
        color: var(--destructive);
      }
      .field-hint {
        font-size: 0.78rem;
        color: var(--muted-foreground);
      }
    `,
  ],
})
export class FormFieldComponent implements ControlValueAccessor, OnInit {
  id = input.required<string>();
  label = input<string>('');
  placeholder = input<string>('');
  type = input<'text' | 'email' | 'password' | 'number'>('text');
  required = input(false);
  disabled = input(false);
  hint = input<string>('');
  error = input<string>('');
  // Optional: form-level error keys to check (e.g., ['passwordMismatch'] for confirm password)
  formErrorKeys = input<string[]>([]);
  errorId = computed(() => `${this.id()}-error`);

  private readonly injector = inject(Injector);
  private ngControl: NgControl | null = null;

  value = signal('');
  showPassword = signal(false);
  private onChange = (v: string): void => {
    void v;
  };
  private onTouched = (): void => {
    void 0;
  };

  private formControlDisabled = signal(false);
  isDisabled = computed(() => this.disabled() || this.formControlDisabled());

  ngOnInit(): void {
    this.ngControl = this.injector.get(NgControl, null, { optional: true, self: true });
  }

  computedError = computed(() => {
    // Explicit error input takes precedence (server-side errors)
    const explicitError = this.error();
    if (explicitError) return explicitError;

    const control = this.ngControl?.control;
    if (!control) return '';

    const form = control.parent as FormGroup | null;
    const formErrorKeys = this.formErrorKeys();

    // Check if control or any form-related field has been interacted with
    const interacted = control.touched || control.dirty;
    let anyFormFieldInteracted = interacted;
    if (form && formErrorKeys.length > 0) {
      anyFormFieldInteracted =
        formErrorKeys.some((key) => {
          const otherControl = form.get(key);
          return otherControl?.touched || otherControl?.dirty;
        }) || interacted;
    }

    // Check form-level errors if any related field has been interacted with
    if (form && formErrorKeys.length > 0 && anyFormFieldInteracted) {
      for (const key of formErrorKeys) {
        if (form.errors?.[key]) {
          if (key === 'passwordMismatch') return 'Passwords do not match';
          return `${this.getFieldLabel()} is invalid`;
        }
      }
    }

    // If not interacted, don't show any errors
    if (!interacted) {
      return '';
    }

    // Check control-level validation errors
    if (control.errors) {
      if (control.errors['required']) return `${this.getFieldLabel()} é obrigatório(a)`;
      if (control.errors['minlength'])
        return `${this.getFieldLabel()} deve ter pelo menos ${control.errors['minlength'].requiredLength} caracteres`;
      if (control.errors['maxlength'])
        return `${this.getFieldLabel()} deve ter no máximo ${control.errors['maxlength'].requiredLength} caracteres`;
      if (control.errors['email']) return 'Informe um e-mail válido';
      if (control.errors['pattern']) return `${this.getFieldLabel()} é inválido(a)`;
      if (control.errors['min'])
        return `${this.getFieldLabel()} deve ser no mínimo ${control.errors['min'].min}`;
      if (control.errors['max'])
        return `${this.getFieldLabel()} deve ser no máximo ${control.errors['max'].max}`;
      return 'Invalid value';
    }

    // Check form-level errors again (in case control has no errors but form does)
    if (form && formErrorKeys.length > 0 && anyFormFieldInteracted) {
      for (const key of formErrorKeys) {
        if (form.errors?.[key]) {
          if (key === 'passwordMismatch') return 'Passwords do not match';
          return `${this.getFieldLabel()} is invalid`;
        }
      }
    }

    return '';
  });

  private getFormLevelError(control: AbstractControl): string | null {
    const form = control.parent as FormGroup | null;
    if (!form) return null;

    const errorKeys = this.formErrorKeys();
    for (const key of errorKeys) {
      if (form.errors?.[key]) {
        // Custom messages for known form-level errors
        if (key === 'passwordMismatch') return 'As senhas não coincidem';
        return `${this.getFieldLabel()} é inválido(a)`;
      }
    }
    return null;
  }

  private getFieldLabel(): string {
    return this.label() || this.placeholder() || 'Este campo';
  }

  onInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.value.set(input.value);
    this.onChange(input.value);
  }

  onBlur(): void {
    this.onTouched();
  }

  togglePassword(): void {
    this.showPassword.update((v) => !v);
  }

  writeValue(value: string): void {
    this.value.set(value ?? '');
  }

  registerOnChange(fn: (v: string) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  setDisabledState(disabled: boolean): void {
    this.formControlDisabled.set(disabled);
  }
}
