import {
  Component,
  input,
  output,
  signal,
  computed,
  effect,
  inject,
  OnInit,
  OnDestroy,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ButtonComponent } from '../button/button.component';
import { Subject, takeUntil } from 'rxjs';
import { trigger, transition, style, animate } from '@angular/animations';

@Component({
  selector: 'app-review-form',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, ButtonComponent],
  animations: [
    trigger('formEnter', [
      transition(':enter', [
        style({ opacity: 0, transform: 'translateY(-8px)' }),
        animate('150ms ease-out', style({ opacity: 1, transform: 'translateY(0)' })),
      ]),
    ]),
  ],
  template: `
    <form [formGroup]="form" (ngSubmit)="onSubmit()" class="review-form" @formEnter>
      <fieldset class="rating-field">
        <legend>Avaliação</legend>
        <!-- eslint-disable-next-line @angular-eslint/template/interactive-supports-focus -->
        <div
          class="star-rating"
          role="radiogroup"
          aria-label="Selecione a avaliação"
          (keydown)="onKeyDown($event)"
        >
          @for (star of stars; track star) {
            <button
              type="button"
              class="star-btn"
              [class.filled]="star <= rating()"
              [class.hovered]="star <= hoverRating()"
              [class.focused]="star === focusedStar()"
              (click)="setRating(star)"
              (mouseenter)="hoverRating.set(star)"
              (mouseleave)="hoverRating.set(0)"
              (focus)="focusedStar.set(star)"
              (blur)="focusedStar.set(0)"
              [attr.aria-label]="'Estrela ' + star + ' de 5'"
              [attr.aria-checked]="star === rating()"
              role="radio"
              tabindex="0"
            >
              <span class="icon icon-star" aria-hidden="true"></span>
            </button>
          }
        </div>
        @if (form.get('rating')?.invalid && form.get('rating')?.touched) {
          <div class="field-error">Selecione uma avaliação</div>
        }
      </fieldset>

      <div class="description-field">
        <label for="review-description" class="field-label">
          Sua avaliação
          <span class="required" aria-hidden="true">*</span>
        </label>
        <textarea
          id="review-description"
          formControlName="description"
          placeholder="Conte sua experiência com este produto..."
          rows="4"
          class="field-textarea"
          [attr.aria-invalid]="form.get('description')?.invalid && form.get('description')?.touched"
          [class.error]="form.get('description')?.invalid && form.get('description')?.touched"
        ></textarea>
        @if (form.get('description')?.invalid && form.get('description')?.touched) {
          <div class="field-error">{{ descriptionError() }}</div>
        }
        <div class="field-hint">Máximo 200 caracteres ({{ descriptionLength() }}/200)</div>
      </div>

      <div class="form-actions">
        <app-button
          label="Cancelar"
          variant="secondary"
          type="button"
          (clicked)="cancelled.emit()"
        />
        <app-button
          label="Enviar avaliação"
          variant="primary"
          type="submit"
          [loading]="submitting()"
          [disabled]="form.invalid"
        />
      </div>
    </form>
  `,
  styles: [
    `
      .review-form {
        display: flex;
        flex-direction: column;
        gap: 1.5rem;
        padding: 1.5rem;
        border: 1px solid var(--color-border);
        border-radius: 1rem;
        background: var(--color-surface-muted);
      }

      .rating-field {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
      }

      .rating-field legend {
        font-size: 0.875rem;
        font-weight: 600;
        color: var(--color-text);
      }

      .star-rating {
        display: flex;
        gap: 0.375rem;
      }

      .star-btn {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 2.75rem;
        height: 2.75rem;
        padding: 0;
        border: 2px solid transparent;
        border-radius: 0.5rem;
        background: var(--color-surface);
        color: var(--color-text-muted);
        cursor: pointer;
        transition:
          color 0.15s,
          border-color 0.15s,
          background 0.15s,
          transform 0.1s;
      }

      .star-btn:hover:not(:disabled),
      .star-btn.hovered:not(:disabled) {
        color: #f59e0b;
        background: color-mix(in oklab, #f59e0b 10%, var(--color-surface));
        transform: scale(1.1);
      }

      .star-btn.filled {
        color: #f59e0b;
      }

      .star-btn.focused {
        outline: none;
        box-shadow: 0 0 0 2px var(--color-focus);
        border-color: var(--color-focus);
      }

      .star-btn:active:not(:disabled) {
        transform: scale(0.95);
      }

      .star-btn .icon {
        width: 1.5rem;
        height: 1.5rem;
      }

      .field-error {
        font-size: 0.75rem;
        color: var(--destructive);
      }

      .description-field {
        display: flex;
        flex-direction: column;
        gap: 0.4rem;
      }

      .field-label {
        font-size: 0.8125rem;
        font-weight: 600;
        color: var(--color-text);
      }

      .required {
        color: var(--destructive);
        margin-left: 0.25rem;
      }

      .field-textarea {
        width: 100%;
        min-height: 7.5rem;
        padding: 0.65rem 0.9rem;
        font: inherit;
        font-size: 0.9375rem;
        color: var(--color-text);
        background: color-mix(in oklab, var(--color-background) 65%, var(--color-surface));
        border: 1px solid var(--color-border);
        border-radius: var(--radius-md);
        resize: vertical;
        transition:
          border-color var(--transition-fast),
          box-shadow var(--transition-fast),
          background-color var(--transition-fast);
      }

      .field-textarea::placeholder {
        color: var(--color-text-muted);
        opacity: 0.8;
      }

      .field-textarea:hover:not(:disabled) {
        border-color: color-mix(in oklab, var(--color-text-muted) 35%, var(--color-border));
      }

      .field-textarea:focus {
        outline: none;
        border-color: var(--color-accent);
        box-shadow: 0 0 0 3px var(--color-primary-bg);
        background: var(--color-background);
      }

      .field-textarea:disabled {
        background: var(--color-muted);
        color: var(--color-text-muted);
        cursor: not-allowed;
      }

      .field-textarea.error {
        border-color: var(--destructive);
      }

      .field-textarea.error:focus {
        box-shadow: 0 0 0 3px var(--color-danger-bg);
      }

      .field-hint {
        font-size: 0.78rem;
        color: var(--color-text-muted);
      }

      .form-actions {
        display: flex;
        justify-content: flex-end;
        gap: 0.75rem;
        margin-top: 0.5rem;
      }

      @media (max-width: 640px) {
        .review-form {
          padding: 1rem;
        }
        .star-btn {
          width: 2.25rem;
          height: 2.25rem;
        }
        .star-btn .icon {
          width: 1.25rem;
          height: 1.25rem;
        }
      }
    `,
  ],
})
export class ReviewFormComponent implements OnInit, OnDestroy {
  private readonly fb = inject(FormBuilder);
  private readonly destroy$ = new Subject<void>();

  readonly productId = input.required<string>();
  readonly submitting = input(false);
  readonly disabled = input(false);

  readonly submitted = output<{ description: string; rating: number }>();
  readonly cancelled = output<void>();

  readonly rating = signal(0);
  readonly hoverRating = signal(0);
  readonly focusedStar = signal(0);
  readonly descriptionLength = signal(0);

  readonly stars = [1, 2, 3, 4, 5] as const;

  readonly form = this.fb.nonNullable.group({
    rating: [0 as number, [Validators.required, Validators.min(1), Validators.max(5)]],
    description: ['', [Validators.required, Validators.minLength(10), Validators.maxLength(200)]],
  });

  readonly descriptionError = computed(() => {
    const control = this.form.get('description');
    if (!control?.touched || !control.errors) return '';
    if (control.errors['required']) return 'A descrição é obrigatória';
    if (control.errors['minlength']) return 'A descrição deve ter pelo menos 10 caracteres';
    if (control.errors['maxlength']) return 'A descrição deve ter no máximo 200 caracteres';
    return '';
  });

  private readonly _disabledEffect = effect(() => {
    if (this.disabled()) {
      this.form.disable();
    } else {
      this.form.enable();
    }
  });

  ngOnInit(): void {
    this.form
      .get('description')
      ?.valueChanges.pipe(takeUntil(this.destroy$))
      .subscribe((value) => {
        this.descriptionLength.set(value?.length ?? 0);
      });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  setRating(value: number): void {
    this.rating.set(value);
    this.form.patchValue({ rating: value });
    this.focusedStar.set(value);
  }

  onKeyDown(event: KeyboardEvent): void {
    const currentRating = this.rating();
    let newRating: number;

    switch (event.key) {
      case 'ArrowRight':
        event.preventDefault();
        newRating = Math.min(5, currentRating + 1);
        break;
      case 'ArrowLeft':
        event.preventDefault();
        newRating = Math.max(1, currentRating - 1);
        break;
      case 'Home':
        event.preventDefault();
        newRating = 1;
        break;
      case 'End':
        event.preventDefault();
        newRating = 5;
        break;
      default:
        return;
    }

    this.setRating(newRating);
  }

  onSubmit(): void {
    if (this.form.valid) {
      this.submitted.emit({
        description: this.form.value.description!,
        rating: this.form.value.rating!,
      });
    } else {
      this.form.markAllAsTouched();
    }
  }
}
