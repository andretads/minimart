import {
  Component,
  inject,
  signal,
  computed,
  OnInit,
  OnDestroy,
  WritableSignal,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  FormBuilder,
  ReactiveFormsModule,
  Validators,
  ValidatorFn,
  AbstractControl,
} from '@angular/forms';
import { Subject, takeUntil } from 'rxjs';
import { CustomerApiService } from '../customer-api.service';
import { AvatarApiService } from '../avatar-api.service';
import { Customer, GetCustomerByUserIdResponse } from '@shared/models/customer.models';
import { Avatar } from '@shared/models/avatar.models';
import { Address, CreateAddressRequest, UpdateAddressRequest } from '@shared/models/address.models';
import { NotificationsApiService } from '@features/notifications/notifications-api.service';
import { ButtonComponent } from '@shared/components/button/button.component';
import { FormFieldComponent } from '@shared/components/form-field/form-field.component';
import { EmptyStateComponent } from '@shared/components/empty-state/empty-state.component';
import { ErrorStateComponent } from '@shared/components/error-state/error-state.component';
import { PageShellComponent } from '@shared/components/page-shell/page-shell.component';
import { SkeletonComponent } from '@shared/components/skeleton/skeleton.component';
import { AuthStore } from '@core/auth/auth.store';
import { AuthService } from '@core/auth/auth.service';
import { ToastService } from '@core/toast/toast.service';
import { ConfirmDialogComponent } from '@shared/components/confirm-dialog/confirm-dialog.component';

type NotificationPreferenceKey = 'emailEnabled' | 'smsEnabled' | 'pushEnabled' | 'marketingEnabled';

interface NotificationSetting {
  key: NotificationPreferenceKey;
  label: string;
  description: string;
  enabled: WritableSignal<boolean>;
}

const passwordMatchValidator: ValidatorFn = (control: AbstractControl) => {
  const newPassword = control.get('newPassword');
  const confirmNewPassword = control.get('confirmNewPassword');

  if (!newPassword || !confirmNewPassword) {
    return null;
  }

  return newPassword.value === confirmNewPassword.value ? null : { passwordMismatch: true };
};

@Component({
  selector: 'app-profile',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    ButtonComponent,
    FormFieldComponent,
    EmptyStateComponent,
    ErrorStateComponent,
    PageShellComponent,
    SkeletonComponent,
    ConfirmDialogComponent,
  ],
  template: `
    <app-page-shell title="Minha conta">
      <div class="profile-page">
        @if (loading()) {
          <div class="profile-skeleton" aria-busy="true">
            <app-skeleton variant="rectangular" width="100%" height="10rem" radius="1.25rem" />
            <app-skeleton variant="rectangular" width="100%" height="24rem" radius="1.25rem" />
          </div>
        } @else if (error() && !restricted()) {
          <app-error-state
            title="Falha ao carregar perfil"
            [message]="error()!"
            retryLabel="Recarregar"
            (retry)="loadProfile()"
          />
        } @else if (restricted()) {
          <app-empty-state
            title="Acesso ao perfil restrito"
            message="Your account role does not have permission to view profile details. This is a backend limitation — only admin and support roles can access this page. Contact support if you believe this is an error."
            icon="lock"
          />
        } @else if (!customer()) {
          <app-empty-state
            title="Perfil não encontrado"
            message="Nenhum perfil de cliente associado à sua conta."
            icon="user"
          />
        } @else {
          <div class="profile-layout">
            <aside class="profile-sidebar">
              <div class="profile-card">
                <button
                  type="button"
                  class="avatar-button"
                  (click)="avatarInput.click()"
                  [disabled]="avatarUploading()"
                  aria-label="Alterar foto do perfil"
                >
                  @if (avatarUrl()) {
                    <img
                      class="avatar-image"
                      [src]="avatarUrl()!"
                      [alt]="'Foto de perfil de ' + customer()?.displayName"
                      (error)="onAvatarImageError()"
                    />
                  } @else {
                    <span class="avatar-initial">{{
                      customer()!.displayName.charAt(0).toUpperCase()
                    }}</span>
                  }
                  @if (avatarUploading()) {
                    <span class="avatar-overlay" aria-hidden="true">
                      <span class="spinner" aria-hidden="true"></span>
                    </span>
                  } @else {
                    <span class="avatar-edit" aria-hidden="true">
                      <span class="icon icon-camera" aria-hidden="true"></span>
                    </span>
                  }
                </button>
                <input
                  #avatarInput
                  type="file"
                  accept="image/jpeg,image/png,image/webp,image/gif"
                  hidden
                  (change)="onAvatarSelected($event)"
                />
                <h2>{{ customer()!.displayName }}</h2>
                <p class="user-id">{{ customer()!.userId }}</p>
                <span class="status-pill" [class.off]="customer()!.status !== 'active'">{{
                  customer()!.status === 'active' ? 'Ativo' : 'Desativado'
                }}</span>
                <div class="member-since">
                  <span>Membro desde</span>
                  <strong>{{ formatDate(customer()!.createdAt) }}</strong>
                </div>
              </div>

              <nav class="profile-nav" aria-label="Seções do perfil">
                <button
                  type="button"
                  [class.active]="activeTab() === 'account'"
                  (click)="setTab('account')"
                >
                  <span class="icon icon-user" aria-hidden="true"></span>
                  <span>Configurações da conta</span>
                  <span class="icon icon-sm icon-chevron-right nav-arrow" aria-hidden="true"></span>
                </button>
                <button
                  type="button"
                  [class.active]="activeTab() === 'security'"
                  (click)="setTab('security')"
                >
                  <span class="icon icon-shield" aria-hidden="true"></span>
                  <span>Segurança</span>
                  <span class="icon icon-sm icon-chevron-right nav-arrow" aria-hidden="true"></span>
                </button>
                <button
                  type="button"
                  [class.active]="activeTab() === 'addresses'"
                  (click)="setTab('addresses')"
                >
                  <span class="icon icon-map-pin" aria-hidden="true"></span>
                  <span>Endereços</span>
                  <span class="icon icon-sm icon-chevron-right nav-arrow" aria-hidden="true"></span>
                </button>
                <button
                  type="button"
                  [class.active]="activeTab() === 'notifications'"
                  (click)="setTab('notifications')"
                >
                  <span class="icon icon-bell" aria-hidden="true"></span>
                  <span>Notificações</span>
                  <span class="icon icon-sm icon-chevron-right nav-arrow" aria-hidden="true"></span>
                </button>
              </nav>
            </aside>

            <section class="profile-main">
              @if (activeTab() === 'account') {
                <header class="panel-head">
                  <span class="eyebrow">Seu perfil</span>
                  <h2>Configurações da conta</h2>
                  <p>Gerencie as informações e preferências da sua conta.</p>
                </header>

                <form [formGroup]="form" (ngSubmit)="save()" class="profile-form">
                  <div class="form-section">
                    <h3>Informações pessoais</h3>
                    <div class="field-grid">
                      <div class="span-2">
                        <app-form-field
                          id="displayName"
                          formControlName="displayName"
                          label="Nome de exibição"
                          type="text"
                          [placeholder]="'Digite seu nome de exibição'"
                          [required]="true"
                          [hint]="'Máximo de 200 caracteres'"
                          [error]="displayNameError()"
                          [disabled]="saving()"
                        />
                      </div>
                      <app-form-field
                        id="email"
                        formControlName="email"
                        label="Endereço de e-mail"
                        type="email"
                        [placeholder]="'you@example.com'"
                        [disabled]="true"
                        [hint]="'Fale com o suporte para alterar o e-mail'"
                      />
                      <app-form-field
                        id="username"
                        formControlName="username"
                        label="Nome de usuário"
                        type="text"
                        [placeholder]="'Seu nome de usuário'"
                        [disabled]="true"
                        [hint]="'Não pode ser alterado'"
                      />
                    </div>
                  </div>

                  <div class="form-section">
                    <h3>Detalhes da conta</h3>
                    <dl class="info-list">
                      <div>
                        <dt>ID do cliente</dt>
                        <dd>{{ customer()!.id }}</dd>
                      </div>
                      <div>
                        <dt>Status da conta</dt>
                        <dd class="capitalize">
                          {{ customer()!.status === 'active' ? 'Ativo' : 'Desativado' }}
                        </dd>
                      </div>
                      <div>
                        <dt>Membro desde</dt>
                        <dd>{{ formatDate(customer()!.createdAt) }}</dd>
                      </div>
                      <div>
                        <dt>Versão do perfil</dt>
                        <dd>{{ customer()!.version }}</dd>
                      </div>
                    </dl>
                  </div>

                  <div class="form-actions">
                    <app-button
                      type="submit"
                      label="Salvar alterações"
                      variant="primary"
                      [loading]="saving()"
                      [disabled]="form.invalid || form.pristine || saving()"
                    />
                    <app-button
                      type="button"
                      variant="ghost"
                      label="Cancelar"
                      (clicked)="resetForm()"
                      [disabled]="form.pristine || saving()"
                    />
                    @if (saveSuccess()) {
                      <span class="save-note"
                        ><span class="icon icon-sm icon-check-circle" aria-hidden="true"></span>
                        Alterações salvas</span
                      >
                    }
                  </div>
                </form>
              }

              @if (activeTab() === 'security') {
                <header class="panel-head">
                  <span class="eyebrow">Proteção</span>
                  <h2>Segurança</h2>
                  <p>Gerencie sua senha e configurações de segurança.</p>
                </header>

                <div class="section-card">
                  <h3>Alterar senha</h3>
                  <p class="section-sub">
                    Use uma senha forte que você não reutilize em outros lugares.
                  </p>
                  <form
                    [formGroup]="passwordForm"
                    (ngSubmit)="changePassword()"
                    class="profile-form"
                  >
                    <app-form-field
                      id="currentPassword"
                      formControlName="currentPassword"
                      label="Senha atual"
                      type="password"
                      [placeholder]="'Digite a senha atual'"
                      [required]="true"
                      [error]="currentPasswordServerError()"
                      [disabled]="changingPassword()"
                    />
                    <div class="field-grid">
                      <app-form-field
                        id="newPassword"
                        formControlName="newPassword"
                        label="Nova senha"
                        type="password"
                        [placeholder]="'Crie uma nova senha'"
                        [required]="true"
                        [hint]="'6-72 caracteres'"
                        [disabled]="changingPassword()"
                      />
                      <app-form-field
                        id="confirmNewPassword"
                        formControlName="confirmNewPassword"
                        label="Confirmar nova senha"
                        type="password"
                        [placeholder]="'Confirme a nova senha'"
                        [required]="true"
                        [formErrorKeys]="['passwordMismatch']"
                        [disabled]="changingPassword()"
                      />
                    </div>
                    <div class="form-actions">
                      <app-button
                        type="submit"
                        label="Atualizar senha"
                        variant="primary"
                        [loading]="changingPassword()"
                        [disabled]="passwordForm.invalid || changingPassword()"
                      />
                    </div>
                  </form>
                </div>
              }

              @if (activeTab() === 'addresses') {
                <header class="panel-head panel-head-row">
                  <div>
                    <span class="eyebrow">Entrega</span>
                    <h2>Endereços</h2>
                    <p>Gerencie seus endereços de entrega e cobrança.</p>
                  </div>
                  <app-button
                    variant="primary"
                    size="sm"
                    label="Adicionar endereço"
                    (clicked)="addAddress()"
                  />
                </header>

                @if (editing()) {
                  <div class="address-editor-panel" role="region" aria-label="Address form">
                    <div class="editor-header">
                      <h3>{{ editingAddressId() ? 'Editar endereço' : 'Novo endereço' }}</h3>
                      <app-button
                        variant="ghost"
                        size="sm"
                        label="Fechar"
                        (clicked)="cancelEdit()"
                        aria-label="Fechar editor"
                      />
                    </div>
                    <form [formGroup]="addressForm" (ngSubmit)="saveAddress()" class="address-form">
                      <div class="field-grid">
                        <div class="space-y-2">
                          <label class="form-label" for="label">Identificação</label>
                          <app-form-field
                            id="label"
                            formControlName="label"
                            type="text"
                            [placeholder]="'Casa, Trabalho…'"
                            [required]="true"
                            [hint]="'ex.: Casa, Trabalho'"
                            [disabled]="savingAddress()"
                          />
                        </div>
                        <div class="space-y-2">
                          <label class="form-label" for="fullName">Nome completo</label>
                          <app-form-field
                            id="fullName"
                            formControlName="fullName"
                            type="text"
                            [placeholder]="'Maria Silva'"
                            [required]="true"
                            [disabled]="savingAddress()"
                          />
                        </div>
                        <div class="space-y-2" style="grid-column: 1 / -1;">
                          <label class="form-label" for="street">Endereço</label>
                          <app-form-field
                            id="street"
                            formControlName="street"
                            type="text"
                            [placeholder]="'Av. Paulista, 1000 — apto 9'"
                            [required]="true"
                            [disabled]="savingAddress()"
                          />
                        </div>
                        <div class="space-y-2">
                          <label class="form-label" for="city">Cidade</label>
                          <app-form-field
                            id="city"
                            formControlName="city"
                            type="text"
                            [placeholder]="'São Paulo'"
                            [required]="true"
                            [disabled]="savingAddress()"
                          />
                        </div>
                        <div class="field-grid" style="grid-column: 1 / -1;">
                          <div class="space-y-2">
                            <label class="form-label" for="state">Estado (UF)</label>
                            <app-form-field
                              id="state"
                              formControlName="state"
                              type="text"
                              [placeholder]="'SP'"
                              [required]="true"
                              [disabled]="savingAddress()"
                            />
                          </div>
                          <div class="space-y-2">
                            <label class="form-label" for="zip">CEP</label>
                            <app-form-field
                              id="zip"
                              formControlName="zip"
                              type="text"
                              [placeholder]="'01310-100'"
                              [required]="true"
                              [disabled]="savingAddress()"
                            />
                          </div>
                        </div>
                        <div class="space-y-2">
                          <label class="form-label" for="country">País</label>
                          <app-form-field
                            id="country"
                            formControlName="country"
                            type="text"
                            [placeholder]="'BR'"
                            [required]="true"
                            [disabled]="savingAddress()"
                          />
                        </div>
                        <div class="space-y-2">
                          <label class="form-label" for="phone">Telefone (opcional)</label>
                          <app-form-field
                            id="phone"
                            formControlName="phone"
                            type="text"
                            [placeholder]="'(11) 99999-9999'"
                            [disabled]="savingAddress()"
                          />
                        </div>
                      </div>
                      <div class="form-checkbox">
                        <label class="checkbox-label">
                          <input
                            type="checkbox"
                            formControlName="isDefault"
                            [disabled]="savingAddress()"
                          />
                          <span>Definir como endereço padrão</span>
                        </label>
                      </div>
                      <div class="form-actions">
                        <app-button
                          type="submit"
                          variant="primary"
                          label="Salvar endereço"
                          [loading]="savingAddress()"
                          [disabled]="addressForm.invalid || savingAddress()"
                        >
                          @if (savingAddress()) {
                            <span class="spinner" aria-hidden="true"></span> Saving…
                          } @else {
                            <span class="icon icon-sm icon-check" aria-hidden="true"></span> Salvar
                            endereço
                          }
                        </app-button>
                        <app-button
                          type="button"
                          variant="ghost"
                          label="Cancelar"
                          (clicked)="cancelEdit()"
                          [disabled]="savingAddress()"
                        />
                      </div>
                    </form>
                  </div>
                }

                @if (!editing() && addresses().length === 0) {
                  <div class="empty-panel">
                    <span class="empty-icon"
                      ><span class="icon icon-xl icon-map-pin" aria-hidden="true"></span
                    ></span>
                    <h3>Nenhum endereço ainda</h3>
                    <p>Adicione seu primeiro endereço para agilizar o checkout.</p>
                    <app-button
                      variant="primary"
                      label="Adicionar endereço"
                      (clicked)="addAddress()"
                    />
                  </div>
                } @else if (!editing()) {
                  <div class="addresses-grid">
                    @for (address of addresses(); track address.id) {
                      <article class="address-card" [class.default]="address.isDefault">
                        <div class="address-header">
                          <h4>{{ address.label }}</h4>
                          @if (address.isDefault) {
                            <span class="default-badge">Padrão</span>
                          }
                        </div>
                        <p>
                          {{ address.fullName }}<br />{{ address.street }}<br />{{ address.city }},
                          {{ address.state }} {{ address.zip }}<br />{{ address.country }}
                        </p>
                        @if (address.phone) {
                          <p class="address-phone">{{ address.phone }}</p>
                        }
                        <div class="address-actions">
                          <app-button
                            variant="ghost"
                            size="sm"
                            label="Editar"
                            (clicked)="editAddress(address)"
                          />
                          @if (!address.isDefault) {
                            <app-button
                              variant="outline"
                              size="sm"
                              label="Definir como padrão"
                              (clicked)="setDefaultAddress(address)"
                            />
                          }
                          <app-button
                            variant="danger"
                            size="sm"
                            label="Excluir"
                            (clicked)="requestDeleteAddress(address)"
                          />
                        </div>
                      </article>
                    }
                  </div>
                }
              }

              @if (activeTab() === 'notifications') {
                <header class="panel-head">
                  <span class="eyebrow">Preferências</span>
                  <h2>Notificações</h2>
                  <p>Escolha como quer receber novidades sobre pedidos e sua conta.</p>
                </header>

                @if (loadingNotifications()) {
                  <div
                    class="settings-list"
                    role="status"
                    aria-busy="true"
                    aria-label="Carregando preferências de notificação"
                  >
                    @for (row of notificationSkeletonRows; track row) {
                      <div class="setting-row">
                        <div
                          class="setting-copy"
                          style="display: flex; flex-direction: column; gap: 0.4rem;"
                        >
                          <app-skeleton variant="text" width="7rem" height="0.875rem" />
                          <app-skeleton variant="text" width="13rem" height="0.75rem" />
                        </div>
                        <app-skeleton
                          variant="rectangular"
                          width="2.75rem"
                          height="1.5rem"
                          radius="999px"
                        />
                      </div>
                    }
                  </div>
                } @else if (notificationsError()) {
                  <app-error-state
                    title="Falha ao carregar preferências de notificação"
                    [message]="notificationsError()!"
                    retryLabel="Tentar novamente"
                    (retry)="loadNotificationPreferences()"
                  />
                } @else {
                  <div class="settings-list" [attr.aria-busy]="savingNotifications()">
                    @for (setting of notificationSettings(); track setting.key) {
                      <div class="setting-row">
                        <div class="setting-copy">
                          <h4>{{ setting.label }}</h4>
                          <p>{{ setting.description }}</p>
                        </div>
                        <label class="switch">
                          <input
                            type="checkbox"
                            role="switch"
                            [attr.aria-label]="setting.label"
                            [attr.aria-checked]="setting.enabled()"
                            [checked]="setting.enabled()"
                            (change)="toggleNotification(setting.key, $event)"
                            [disabled]="loadingNotifications() || savingNotifications()"
                          />
                          <span class="slider"></span>
                        </label>
                      </div>
                    }
                  </div>
                }
              }
            </section>
          </div>
        }
      </div>
    </app-page-shell>

    <app-confirm-dialog
      [open]="confirmDeleteAddress()"
      title="Excluir endereço"
      message="Tem certeza que deseja excluir este endereço? Esta ação não pode ser desfeita."
      confirmLabel="Excluir endereço"
      confirmVariant="danger"
      [confirmLoading]="deletingAddress()"
      (close)="confirmDeleteAddress.set(false)"
      (confirm)="deleteAddress()"
    />
  `,
  styles: [
    `
      .profile-page {
        max-width: 74rem;
        margin: 0 auto;
      }

      .profile-layout {
        display: grid;
        grid-template-columns: 1fr;
        gap: 1.75rem;
        align-items: start;
      }
      @media (min-width: 992px) {
        .profile-layout {
          grid-template-columns: 17rem minmax(0, 1fr);
        }
      }
      .profile-sidebar {
        display: flex;
        flex-direction: column;
        gap: 1rem;
      }
      @media (min-width: 992px) {
        .profile-sidebar {
          position: sticky;
          top: 6rem;
        }
      }

      .profile-card {
        padding: 1.5rem;
        border: 1px solid var(--color-border);
        border-radius: 1.25rem;
        background: var(--color-surface);
        box-shadow: var(--shadow-xs);
        text-align: center;
      }
      .avatar-button {
        position: relative;
        display: grid;
        place-items: center;
        width: 4.5rem;
        height: 4.5rem;
        margin: 0 auto 1rem;
        border-radius: 1.25rem;
        border: 2px solid var(--color-border);
        background: linear-gradient(
          145deg,
          var(--accent),
          color-mix(in oklab, var(--accent) 55%, #0f766e)
        );
        color: var(--accent-foreground);
        font-size: 1.55rem;
        font-weight: 750;
        box-shadow: 0 14px 28px -16px color-mix(in oklab, var(--accent) 70%, transparent);
        cursor: pointer;
        overflow: hidden;
        transition:
          border-color 0.2s,
          box-shadow 0.2s;
      }
      .avatar-button:hover,
      .avatar-button:focus {
        border-color: var(--accent);
        box-shadow:
          0 0 0 3px color-mix(in oklab, var(--accent) 20%, transparent),
          0 14px 28px -16px color-mix(in oklab, var(--accent) 70%, transparent);
        outline: none;
      }
      .avatar-button:disabled {
        cursor: wait;
        opacity: 0.8;
      }
      .avatar-button:disabled:hover {
        border-color: var(--color-border);
        box-shadow: 0 14px 28px -16px color-mix(in oklab, var(--accent) 70%, transparent);
      }
      .avatar-image {
        width: 100%;
        height: 100%;
        object-fit: cover;
        border-radius: 1rem;
      }
      .avatar-initial {
        display: grid;
        place-items: center;
        width: 100%;
        height: 100%;
      }
      .avatar-overlay {
        position: absolute;
        inset: 0;
        display: grid;
        place-items: center;
        background: color-mix(in oklab, var(--color-surface) 85%, transparent);
        border-radius: 1rem;
        z-index: 1;
      }
      .avatar-edit {
        position: absolute;
        bottom: 0.25rem;
        right: 0.25rem;
        display: grid;
        place-items: center;
        width: 1.5rem;
        height: 1.5rem;
        border-radius: 50%;
        background: var(--accent);
        color: var(--accent-foreground);
        box-shadow: 0 2px 6px -2px rgb(0 0 0 / 0.3);
        opacity: 0;
        transition:
          opacity 0.2s,
          transform 0.2s;
        transform: translate(0.25rem, 0.25rem);
      }
      .avatar-button:hover .avatar-edit,
      .avatar-button:focus .avatar-edit {
        opacity: 1;
        transform: translate(0, 0);
      }
      .avatar-button:disabled .avatar-edit {
        opacity: 0;
      }
      .profile-card h2 {
        margin: 0 0 0.2rem;
        font-size: 1.05rem;
        font-weight: 700;
        letter-spacing: -0.02em;
      }
      .user-id {
        margin: 0;
        font-family: var(--font-mono);
        font-size: 0.6875rem;
        color: var(--color-text-muted);
        overflow-wrap: anywhere;
      }
      .status-pill {
        display: inline-flex;
        align-items: center;
        gap: 0.35rem;
        margin-top: 0.75rem;
        padding: 0.28rem 0.65rem;
        border-radius: 999px;
        font-size: 0.6875rem;
        font-weight: 600;
        text-transform: capitalize;
        color: var(--color-accent);
        background: color-mix(in oklab, var(--accent) 11%, transparent);
      }
      .status-pill.off {
        color: var(--color-text-muted);
        background: var(--color-surface-muted);
      }
      .member-since {
        display: flex;
        flex-direction: column;
        gap: 0.15rem;
        margin-top: 1.25rem;
        padding-top: 1rem;
        border-top: 1px solid var(--color-border);
      }
      .member-since span {
        font-size: 0.625rem;
        font-weight: 600;
        letter-spacing: 0.1em;
        text-transform: uppercase;
        color: var(--color-text-muted);
      }
      .member-since strong {
        font-size: 0.75rem;
        font-weight: 600;
      }

      .profile-nav {
        display: flex;
        flex-direction: column;
        padding: 0.5rem;
        border: 1px solid var(--color-border);
        border-radius: 1.1rem;
        background: var(--color-surface);
      }
      .profile-nav button {
        display: grid;
        grid-template-columns: auto 1fr auto;
        align-items: center;
        gap: 0.6rem;
        padding: 0.7rem 0.8rem;
        border: 0;
        border-radius: 0.75rem;
        background: none;
        color: var(--color-text-muted);
        font: inherit;
        font-size: 0.8125rem;
        font-weight: 500;
        text-align: left;
        cursor: pointer;
        transition:
          background 0.2s,
          color 0.2s;
      }
      .profile-nav button:hover {
        background: var(--color-surface-muted);
        color: var(--color-text);
      }
      .profile-nav button.active {
        background: color-mix(in oklab, var(--accent) 11%, transparent);
        color: var(--color-accent);
        font-weight: 600;
      }
      .nav-arrow {
        opacity: 0.3;
      }
      .profile-nav button.active .nav-arrow {
        opacity: 0.8;
      }

      .profile-main {
        min-width: 0;
        padding: clamp(1.25rem, 3vw, 2rem);
        border: 1px solid var(--color-border);
        border-radius: 1.25rem;
        background: var(--color-surface);
        box-shadow: var(--shadow-xs);
      }
      .panel-head {
        margin-bottom: 1.75rem;
        padding-bottom: 1.25rem;
        border-bottom: 1px solid var(--color-border);
      }
      .panel-head h2 {
        margin: 0.15rem 0 0.35rem;
        font-size: clamp(1.4rem, 2.4vw, 1.75rem);
        font-weight: 750;
        letter-spacing: -0.03em;
      }
      .panel-head p {
        margin: 0;
        font-size: 0.875rem;
        color: var(--color-text-muted);
      }
      .panel-head-row {
        display: flex;
        align-items: flex-end;
        justify-content: space-between;
        flex-wrap: wrap;
        gap: 1rem;
      }

      .profile-form {
        display: flex;
        flex-direction: column;
        gap: 1.5rem;
      }
      .form-section h3 {
        margin: 0 0 1rem;
        font-size: 0.9375rem;
        font-weight: 700;
        letter-spacing: -0.01em;
      }
      .field-grid {
        display: grid;
        grid-template-columns: 1fr;
        gap: 1.1rem;
      }
      @media (min-width: 640px) {
        .field-grid {
          grid-template-columns: repeat(2, minmax(0, 1fr));
        }
        .span-2 {
          grid-column: span 2;
        }
      }
      .info-list {
        display: grid;
        gap: 0;
        margin: 0;
      }
      .info-list div {
        display: flex;
        justify-content: space-between;
        gap: 1rem;
        padding: 0.65rem 0;
        border-bottom: 1px solid var(--color-border);
      }
      .info-list div:last-child {
        border-bottom: 0;
      }
      .info-list dt {
        font-size: 0.8125rem;
        color: var(--color-text-muted);
      }
      .info-list dd {
        margin: 0;
        font-size: 0.8125rem;
        font-weight: 500;
        text-align: right;
        overflow-wrap: anywhere;
      }
      .form-actions {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 0.6rem;
        padding-top: 1.25rem;
        border-top: 1px solid var(--color-border);
      }
      .save-note {
        display: inline-flex;
        align-items: center;
        gap: 0.3rem;
        font-size: 0.75rem;
        color: var(--color-accent);
      }

      .section-card {
        margin-bottom: 1rem;
        padding: 1.25rem;
        border: 1px solid var(--color-border);
        border-radius: 1rem;
        background: var(--color-surface-muted);
      }
      .section-card > h3,
      .section-head h3 {
        margin: 0;
        font-size: 0.9375rem;
        font-weight: 700;
      }
      .section-sub {
        margin: 0.25rem 0 1.25rem;
        font-size: 0.8125rem;
        color: var(--color-text-muted);
      }
      .section-head {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        flex-wrap: wrap;
        gap: 1rem;
        margin-bottom: 1rem;
      }
      .section-head .section-sub {
        margin: 0.2rem 0 0;
      }

      .info-banner {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 0.9rem;
        margin: 1rem 0;
        padding: 1.1rem 1.25rem;
        border: 1px solid color-mix(in oklab, var(--accent) 20%, transparent);
        border-radius: 1rem;
        background: color-mix(in oklab, var(--accent) 6%, transparent);
      }
      .info-icon {
        display: grid;
        place-items: center;
        flex: 0 0 auto;
        width: 2.25rem;
        height: 2.25rem;
        border-radius: 0.7rem;
        background: color-mix(in oklab, var(--accent) 13%, transparent);
        color: var(--color-accent);
      }
      .info-copy {
        flex: 1;
        min-width: 12rem;
      }
      .info-copy h4 {
        margin: 0 0 0.15rem;
        font-size: 0.875rem;
        font-weight: 600;
      }
      .info-copy p {
        margin: 0;
        font-size: 0.75rem;
        color: var(--color-text-muted);
      }

      .session-row {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        padding-top: 0.85rem;
        border-top: 1px solid var(--color-border);
      }
      .session-icon {
        display: grid;
        place-items: center;
        flex: 0 0 auto;
        width: 2rem;
        height: 2rem;
        border-radius: 0.6rem;
        background: var(--color-surface);
        color: var(--color-text-muted);
      }
      .session-copy {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 0.1rem;
      }
      .session-copy strong {
        font-size: 0.8125rem;
        font-weight: 600;
      }
      .session-copy span {
        font-size: 0.75rem;
        color: var(--color-text-muted);
      }

      .empty-panel {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.4rem;
        padding: 3rem 1.5rem;
        border: 1px dashed var(--color-border);
        border-radius: 1rem;
        background: var(--color-surface-muted);
        text-align: center;
      }
      .empty-icon {
        display: grid;
        place-items: center;
        width: 3.25rem;
        height: 3.25rem;
        margin-bottom: 0.35rem;
        border-radius: 1rem;
        background: color-mix(in oklab, var(--accent) 11%, transparent);
        color: var(--color-accent);
      }
      .empty-panel h3 {
        margin: 0;
        font-size: 1rem;
        font-weight: 700;
      }
      .empty-panel p {
        margin: 0 0 0.75rem;
        font-size: 0.8125rem;
        color: var(--color-text-muted);
      }

      .addresses-grid {
        display: grid;
        grid-template-columns: 1fr;
        gap: 1rem;
      }
      @media (min-width: 640px) {
        .addresses-grid {
          grid-template-columns: repeat(auto-fill, minmax(16rem, 1fr));
        }
      }
      .address-card {
        position: relative;
        padding: 1.25rem;
        border: 1px solid var(--color-border);
        border-radius: 1rem;
        background: var(--color-surface-muted);
        transition:
          border-color 0.2s,
          box-shadow 0.2s;
      }
      .address-card.default {
        border-color: color-mix(in oklab, var(--accent) 40%, transparent);
        box-shadow: 0 0 0 3px color-mix(in oklab, var(--accent) 8%, transparent);
      }
      .address-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 0.5rem;
        margin-bottom: 0.5rem;
      }
      .address-header h4 {
        margin: 0;
        font-size: 0.9375rem;
        font-weight: 700;
      }
      .default-badge {
        font-size: 0.625rem;
        font-weight: 600;
        padding: 0.15rem 0.5rem;
        border-radius: 999px;
        background: color-mix(in oklab, var(--accent) 11%, transparent);
        color: var(--color-accent);
        text-transform: uppercase;
      }
      .address-card p {
        margin: 0;
        font-size: 0.8125rem;
        line-height: 1.6;
        color: var(--color-text-muted);
      }
      .address-phone {
        margin-top: 0.5rem;
        font-size: 0.8125rem;
        color: var(--color-text-muted);
      }
      .address-actions {
        display: flex;
        flex-wrap: wrap;
        gap: 0.35rem;
        margin-top: 1rem;
      }

      .address-editor-panel {
        margin-bottom: 1.5rem;
        padding: 1.5rem;
        border: 1px solid var(--color-border);
        border-radius: 1rem;
        background: var(--color-surface);
        box-shadow: var(--shadow-xs);
        animation: slideDown 0.2s ease-out;
      }
      @keyframes slideDown {
        from {
          opacity: 0;
          transform: translateY(-10px);
        }
        to {
          opacity: 1;
          transform: translateY(0);
        }
      }
      .editor-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 1.5rem;
      }
      .editor-header h3 {
        margin: 0;
        font-size: 1.1rem;
        font-weight: 700;
      }
      .address-form {
        display: flex;
        flex-direction: column;
        gap: 1.25rem;
      }
      .address-form .field-grid {
        display: grid;
        grid-template-columns: 1fr;
        gap: 1.1rem;
      }
      @media (min-width: 640px) {
        .address-form .field-grid {
          grid-template-columns: repeat(2, minmax(0, 1fr));
        }
      }
      .form-label {
        display: block;
        font-size: 0.8125rem;
        font-weight: 500;
        margin-bottom: 0.35rem;
        color: var(--color-text);
      }
      .form-checkbox {
        margin-top: 0.5rem;
      }
      .checkbox-label {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        font-size: 0.875rem;
        color: var(--color-text);
        cursor: pointer;
      }
      .checkbox-label input {
        width: 1.125rem;
        height: 1.125rem;
        accent-color: var(--color-accent);
      }

      .settings-list {
        overflow: hidden;
        border: 1px solid var(--color-border);
        border-radius: 1rem;
        background: var(--color-surface-muted);
      }
      .setting-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1rem;
        padding: 1.1rem 1.25rem;
        border-bottom: 1px solid var(--color-border);
      }
      .setting-row:last-child {
        border-bottom: 0;
      }
      .setting-copy h4 {
        margin: 0 0 0.15rem;
        font-size: 0.875rem;
        font-weight: 600;
      }
      .setting-copy p {
        margin: 0;
        font-size: 0.8125rem;
        color: var(--color-text-muted);
      }

      .switch {
        position: relative;
        display: inline-block;
        flex: 0 0 auto;
        width: 2.75rem;
        height: 1.5rem;
      }
      .switch input {
        opacity: 0;
        width: 0;
        height: 0;
      }
      .slider {
        position: absolute;
        inset: 0;
        cursor: pointer;
        border-radius: 999px;
        background: var(--color-border);
        transition: background 0.25s;
      }
      .slider::before {
        content: '';
        position: absolute;
        left: 0.1875rem;
        bottom: 0.1875rem;
        width: 1.125rem;
        height: 1.125rem;
        border-radius: 50%;
        background: #fff;
        box-shadow: 0 1px 3px rgb(0 0 0 / 0.2);
        transition: transform 0.25s;
      }
      input:checked + .slider {
        background: var(--color-accent);
      }
      input:checked + .slider::before {
        transform: translateX(1.25rem);
      }
      input:disabled + .slider {
        opacity: 0.5;
        cursor: not-allowed;
      }

      .spinner {
        display: inline-block;
        width: 1rem;
        height: 1rem;
        border: 2px solid currentColor;
        border-right-color: transparent;
        border-radius: 50%;
        animation: spin 0.75s linear infinite;
        margin-right: 0.5rem;
      }
      @keyframes spin {
        to {
          transform: rotate(360deg);
        }
      }

      @media (max-width: 640px) {
        .profile-main {
          padding: 1.1rem;
        }
        .panel-head-row {
          align-items: flex-start;
        }
        .address-form .field-grid {
          grid-template-columns: 1fr;
        }
      }
    `,
  ],
})
export class ProfileComponent implements OnInit, OnDestroy {
  private readonly api = inject(CustomerApiService);
  private readonly avatarApi = inject(AvatarApiService);
  private readonly authService = inject(AuthService);
  private readonly authStore = inject(AuthStore);
  private readonly notificationsApi = inject(NotificationsApiService);
  private readonly toast = inject(ToastService);
  private readonly fb = inject(FormBuilder);
  private readonly destroy$ = new Subject<void>();

  customer = signal<Customer | null>(null);
  loading = signal(true);
  error = signal<string | null>(null);
  saving = signal(false);
  saveSuccess = signal(false);
  displayNameServerError = signal('');

  avatarUrl = signal<string | null>(null);
  avatarUploading = signal(false);
  avatarError = signal<string | null>(null);

  displayNameError = computed(() => {
    const control = this.form.get('displayName');
    if (control?.errors && (control.touched || control.dirty)) {
      if (control.errors['required']) return 'Nome de exibição é obrigatório';
      if (control.errors['maxlength']) return 'Nome de exibição deve ter no máximo 200 caracteres';
    }

    if (this.displayNameServerError()) {
      return this.displayNameServerError();
    }
    return '';
  });
  changingPassword = signal(false);
  currentPasswordServerError = signal('');
  savingNotifications = signal(false);
  loadingNotifications = signal(false);
  notificationsError = signal<string | null>(null);

  addresses = signal<Address[]>([]);
  loadingAddresses = signal(false);
  addressesError = signal<string | null>(null);
  editing = signal(false);
  editingAddressId = signal<string | null>(null);
  savingAddress = signal(false);
  addressFormErrors = signal<Record<string, string | string[]>>({});
  confirmDeleteAddress = signal(false);
  deletingAddress = signal(false);
  addressPendingDelete = signal<Address | null>(null);

  labelError = computed(() => this.getAddressFieldError('label'));
  fullNameError = computed(() => this.getAddressFieldError('fullName'));
  streetError = computed(() => this.getAddressFieldError('street'));
  cityError = computed(() => this.getAddressFieldError('city'));
  stateError = computed(() => this.getAddressFieldError('state'));
  zipError = computed(() => this.getAddressFieldError('zip'));
  countryError = computed(() => this.getAddressFieldError('country'));
  phoneError = computed(() => this.getAddressFieldError('phone'));

  private getAddressFieldError(field: string): string {
    const control = this.addressForm.get(field);
    if (control?.errors && (control.touched || control.dirty)) {
      if (control.errors['required']) return `${this.getFieldLabel(field)} é obrigatório(a)`;
      if (control.errors['maxlength'])
        return `${this.getFieldLabel(field)} deve ter no máximo o tamanho permitido`;
    }

    const errors = this.addressFormErrors();
    const serverError = errors[field];
    if (serverError) {
      return Array.isArray(serverError) ? serverError[0] : serverError;
    }
    return '';
  }

  private getFieldLabel(field: string): string {
    const labels: Record<string, string> = {
      label: 'Etiqueta',
      fullName: 'Nome completo',
      street: 'Logradouro e número',
      city: 'Cidade',
      state: 'UF',
      zip: 'CEP',
      country: 'País',
      phone: 'Telefone',
    };
    return labels[field] ?? field;
  }

  getFirstError(field: string): string {
    const errors = this.addressFormErrors();
    const error = errors[field];
    if (!error) return '';
    return Array.isArray(error) ? error[0] : error;
  }

  readonly notificationSkeletonRows = [0, 1, 2, 3];

  activeTab = signal<'account' | 'security' | 'addresses' | 'notifications'>('account');

  form = this.fb.nonNullable.group({
    displayName: ['', [Validators.required, Validators.maxLength(200)]],
    email: ['', [Validators.email]],
    username: [''],
    version: [0, [Validators.required, Validators.min(1)]],
  });

  passwordForm = this.fb.nonNullable.group(
    {
      currentPassword: ['', [Validators.required]],
      newPassword: ['', [Validators.required, Validators.minLength(6), Validators.maxLength(72)]],
      confirmNewPassword: ['', [Validators.required]],
    },
    { validators: passwordMatchValidator },
  );

  addressForm = this.fb.nonNullable.group({
    label: ['', [Validators.required, Validators.maxLength(50)]],
    fullName: ['', [Validators.required, Validators.maxLength(100)]],
    street: ['', [Validators.required, Validators.maxLength(200)]],
    city: ['', [Validators.required, Validators.maxLength(100)]],
    state: ['', [Validators.required, Validators.maxLength(100)]],
    zip: ['', [Validators.required, Validators.maxLength(20)]],
    country: ['', [Validators.required, Validators.maxLength(100)]],
    phone: ['', [Validators.maxLength(30)]],
    isDefault: [false],
    version: [0, [Validators.required, Validators.min(1)]],
  });

  notificationSettings = signal<NotificationSetting[]>([
    {
      key: 'emailEnabled',
      label: 'Email',
      description: 'Receba notificações por email',
      enabled: signal(true),
    },
    {
      key: 'smsEnabled',
      label: 'SMS',
      description: 'Receba notificações por SMS',
      enabled: signal(false),
    },
    {
      key: 'pushEnabled',
      label: 'Push',
      description: 'Receba notificações push',
      enabled: signal(false),
    },
    {
      key: 'marketingEnabled',
      label: 'Marketing',
      description: 'Receba ofertas e novidades',
      enabled: signal(false),
    },
  ]);

  private notificationPreferencesLoaded = false;

  restricted = computed(() => this.error()?.includes('403') ?? false);

  ngOnInit(): void {
    this.loadProfile();
    this.passwordForm.controls.currentPassword.valueChanges
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => this.currentPasswordServerError.set(''));
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  loadProfile(): void {
    this.loading.set(true);
    this.error.set(null);

    this.api.getByUserId().subscribe({
      next: (response: GetCustomerByUserIdResponse) => {
        const customer: Customer = {
          ...response,
          status: response.status as 'active' | 'deactivated',
        };
        this.customer.set(customer);
        this.loadAvatar(response.userId);

        this.authService.me().subscribe({
          next: (meResponse) => {
            this.form.patchValue({
              displayName: response.displayName,
              email: meResponse.email,
              username: meResponse.username,
              version: response.version,
            });
            this.loading.set(false);
            this.loadAddresses();
          },
          error: () => {
            const storedUsername = this.authStore.user()?.username ?? response.userId;
            this.form.patchValue({
              displayName: response.displayName,
              email: '',
              username: storedUsername,
              version: response.version,
            });
            this.loading.set(false);
            this.loadAddresses();
          },
        });
      },
      error: (err: unknown) => {
        this.error.set((err as { message?: string }).message || 'Erro desconhecido');
        this.loading.set(false);
      },
    });
  }

  loadAvatar(accountId: string): void {
    this.avatarApi.get(accountId).subscribe({
      next: (avatar: Avatar) => {
        if (avatar.imageUrl) {
          this.avatarUrl.set(avatar.imageUrl);
        } else {
          this.avatarUrl.set(null);
        }
      },
      error: (err: unknown) => {
        const httpErr = err as { status?: number; message?: string };
        if (httpErr.status !== 404) {
          console.warn('Failed to load avatar:', httpErr.message);
        }
        this.avatarUrl.set(null);
      },
    });
  }

  onAvatarSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    input.value = '';

    const allowedTypes = ['image/jpeg', 'image/png', 'image/webp', 'image/gif'];
    if (!allowedTypes.includes(file.type)) {
      this.toast.error('Formato de imagem inválido. Use JPEG, PNG, WebP ou GIF.');
      return;
    }
    const maxSize = 5 * 1024 * 1024;
    if (file.size > maxSize) {
      this.toast.error('Imagem muito grande. Tamanho máximo: 5 MB.');
      return;
    }

    this.avatarUploading.set(true);
    this.avatarError.set(null);

    const accountId = this.customer()?.userId ?? this.authStore.userId() ?? '';
    if (!accountId) {
      this.toast.error('Não foi possível identificar a conta.');
      this.avatarUploading.set(false);
      return;
    }

    this.avatarApi.upload(file, accountId).subscribe({
      next: (avatar: Avatar) => {
        if (avatar.imageUrl) {
          this.avatarUrl.set(avatar.imageUrl);
        }
        this.avatarUploading.set(false);
        this.toast.success('Avatar atualizado com sucesso');
      },
      error: (err: unknown) => {
        this.avatarUploading.set(false);
        const httpErr = err as {
          status?: number;
          message?: string;
          fields?: Record<string, string[]>;
        };
        if (httpErr.status === 403) {
          this.toast.error('Você não pode alterar o avatar de outra conta.');
        } else if (httpErr.status === 400) {
          this.toast.error(httpErr.message || 'Dados de upload inválidos.');
        } else if (httpErr.status === 413) {
          this.toast.error('Imagem muito grande. Tamanho máximo: 5 MB.');
        } else {
          this.toast.error(httpErr.message || 'Erro ao fazer upload do avatar.');
        }
      },
    });
  }

  onAvatarImageError(): void {
    this.avatarUrl.set(null);
  }

  loadAddresses(): void {
    this.loadingAddresses.set(true);
    this.addressesError.set(null);
    this.api.getAddresses().subscribe({
      next: (addresses) => {
        this.addresses.set(addresses);
        this.loadingAddresses.set(false);
      },
      error: (err: unknown) => {
        this.addressesError.set(
          (err as { message?: string }).message || 'Falha ao carregar endereços',
        );
        this.loadingAddresses.set(false);
      },
    });
  }

  setTab(tab: 'account' | 'security' | 'addresses' | 'notifications'): void {
    this.activeTab.set(tab);
    if (
      tab === 'notifications' &&
      !this.notificationPreferencesLoaded &&
      !this.loadingNotifications()
    ) {
      this.loadNotificationPreferences();
    }
  }

  save(): void {
    if (this.form.invalid || !this.customer()) return;
    this.saving.set(true);
    this.saveSuccess.set(false);
    this.displayNameServerError.set('');

    const customer = this.customer()!;
    const { displayName, version } = this.form.getRawValue();

    this.api.update(customer.id, { displayName, version }).subscribe({
      next: (updated) => {
        this.customer.set(updated);
        this.form.patchValue({ version: updated.version });
        this.form.markAsPristine();
        this.saving.set(false);
        this.saveSuccess.set(true);
        setTimeout(() => this.saveSuccess.set(false), 3000);
      },
      error: (err: unknown) => {
        this.saving.set(false);
        const httpErr = err as {
          status?: number;
          message?: string;
          fields?: Record<string, string[]>;
        };
        if (httpErr.status === 409) {
          this.displayNameServerError.set(
            'Este perfil foi modificado em outra sessão. Recarregue a página e tente novamente.',
          );
          this.loadProfile();
        } else if (httpErr.status === 400 && httpErr.fields?.['displayName']) {
          this.displayNameServerError.set(
            httpErr.fields['displayName'][0] || 'Nome de exibição inválido',
          );
        } else {
          this.toast.error(httpErr.message || 'Erro desconhecido');
        }
      },
    });
  }

  resetForm(): void {
    const customer = this.customer();
    if (!customer) return;
    this.form.patchValue({
      displayName: customer.displayName,
      email: this.form.value.email,
      username: this.authStore.user()?.username ?? customer.userId,
      version: customer.version,
    });
    this.displayNameServerError.set('');
    this.saveSuccess.set(false);
  }

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString('pt-BR', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  }

  currentPasswordError = computed(() => {
    const control = this.passwordForm.get('currentPassword');
    if (control?.errors && (control.touched || control.dirty)) {
      if (control.errors['required']) return 'Senha atual é obrigatória';
    }
    if (this.currentPasswordServerError()) return this.currentPasswordServerError();
    return '';
  });

  newPasswordError = computed(() => {
    const control = this.passwordForm.get('newPassword');
    if (control?.errors && (control.touched || control.dirty)) {
      if (control.errors['required']) return 'Nova senha é obrigatória';
      if (control.errors['minlength']) return 'A senha deve ter pelo menos 6 caracteres';
      if (control.errors['maxlength']) return 'A senha deve ter no máximo 72 caracteres';
    }
    return '';
  });

  confirmNewPasswordError = computed(() => {
    const control = this.passwordForm.get('confirmNewPassword');
    const newPasswordControl = this.passwordForm.get('newPassword');
    const formErrors = this.passwordForm.errors;
    if (control?.errors && (control.touched || control.dirty)) {
      if (control.errors['required']) return 'Confirme sua senha';
    }
    const interacted =
      control?.touched ||
      control?.dirty ||
      newPasswordControl?.touched ||
      newPasswordControl?.dirty;
    if (formErrors?.['passwordMismatch'] && interacted) {
      return 'As senhas não coincidem';
    }
    return '';
  });

  changePassword(): void {
    if (this.passwordForm.invalid || this.changingPassword()) return;
    this.changingPassword.set(true);
    this.currentPasswordServerError.set('');

    const { currentPassword, newPassword, confirmNewPassword } = this.passwordForm.getRawValue();

    this.authService
      .changePassword({ currentPassword, newPassword, confirmNewPassword })
      .subscribe({
        next: () => {
          this.changingPassword.set(false);
          this.passwordForm.reset();
          this.toast.success('Senha alterada com sucesso');
        },
        error: (err: unknown) => {
          this.changingPassword.set(false);
          const httpErr = err as {
            status?: number;
            message?: string;
            fields?: Record<string, string[]>;
          };
          if (
            (httpErr.status === 400 && httpErr.message?.includes('senha atual incorreta')) ||
            httpErr.message?.includes('current password is incorrect')
          ) {
            this.currentPasswordServerError.set('Senha atual incorreta');
          } else {
            this.toast.error(httpErr.message || 'Erro desconhecido');
          }
        },
      });
  }

  addAddress(): void {
    this.editingAddressId.set(null);
    this.addressForm.reset({
      label: '',
      fullName: '',
      street: '',
      city: '',
      state: '',
      zip: '',
      country: 'BR',
      phone: '',
      isDefault: this.addresses().length === 0,
      version: 1,
    });
    this.addressFormErrors.set({});
    this.editing.set(true);
  }

  editAddress(address: Address): void {
    this.editingAddressId.set(address.id);
    this.addressForm.patchValue({
      label: address.label,
      fullName: address.fullName,
      street: address.street,
      city: address.city,
      state: address.state,
      zip: address.zip,
      country: address.country,
      phone: address.phone || '',
      isDefault: address.isDefault,
      version: address.version,
    });
    this.addressFormErrors.set({});
    this.editing.set(true);
  }

  cancelEdit(): void {
    this.editing.set(false);
    this.editingAddressId.set(null);
    this.addressForm.reset();
    this.addressFormErrors.set({});
  }

  saveAddress(): void {
    if (this.addressForm.invalid) return;
    this.savingAddress.set(true);
    this.addressFormErrors.set({});

    const formValue = this.addressForm.getRawValue();
    const requestData: CreateAddressRequest | UpdateAddressRequest = {
      label: formValue.label,
      fullName: formValue.fullName,
      street: formValue.street,
      city: formValue.city,
      state: formValue.state,
      zip: formValue.zip,
      country: formValue.country,
      phone: formValue.phone || undefined,
      isDefault: formValue.isDefault,
    };

    const isEditing = !!this.editingAddressId();
    const addressId = this.editingAddressId()!;

    if (isEditing) {
      (requestData as UpdateAddressRequest).version = formValue.version;
    }

    const request$ = isEditing
      ? this.api.updateAddress(addressId, requestData as UpdateAddressRequest)
      : this.api.createAddress(requestData as CreateAddressRequest);

    request$.subscribe({
      next: () => {
        this.savingAddress.set(false);
        this.toast.success(isEditing ? 'Endereço atualizado' : 'Endereço criado');
        this.cancelEdit();
        this.loadAddresses();
      },
      error: (err: unknown) => {
        this.savingAddress.set(false);
        const httpErr = err as {
          status?: number;
          message?: string;
          fields?: Record<string, string[]>;
        };
        if (httpErr.status === 409) {
          this.toast.error(
            'Este endereço foi modificado em outra sessão. Recarregue a página e tente novamente.',
          );
          this.loadAddresses();
          this.cancelEdit();
        } else if (httpErr.status === 400 && httpErr.fields) {
          this.addressFormErrors.set(httpErr.fields);
        } else {
          this.toast.error(httpErr.message || 'Erro desconhecido');
        }
      },
    });
  }

  setDefaultAddress(address: Address): void {
    if (address.isDefault) return;
    this.api.setDefaultAddress(address.id).subscribe({
      next: () => {
        this.toast.success('Endereço padrão atualizado');
        this.loadAddresses();
      },
      error: (err: unknown) => {
        const httpErr = err as { status?: number; message?: string };
        this.toast.error(httpErr.message || 'Falha ao definir endereço padrão');
      },
    });
  }

  requestDeleteAddress(address: Address): void {
    this.addressPendingDelete.set(address);
    this.confirmDeleteAddress.set(true);
  }

  deleteAddress(): void {
    const address = this.addressPendingDelete();
    if (!address) return;
    this.deletingAddress.set(true);
    this.api.deleteAddress(address.id).subscribe({
      next: () => {
        this.deletingAddress.set(false);
        this.confirmDeleteAddress.set(false);
        this.addressPendingDelete.set(null);
        this.toast.success('Endereço excluído');
        this.loadAddresses();
      },
      error: (err: unknown) => {
        this.deletingAddress.set(false);
        this.confirmDeleteAddress.set(false);
        this.addressPendingDelete.set(null);
        const httpErr = err as { status?: number; message?: string };
        this.toast.error(httpErr.message || 'Falha ao excluir endereço');
      },
    });
  }

  loadNotificationPreferences(): void {
    this.loadingNotifications.set(true);
    this.notificationsError.set(null);
    this.notificationsApi.getPreferences().subscribe({
      next: (prefs) => {
        this.notificationSettings.update((settings) =>
          settings.map((s) => ({ ...s, enabled: signal(prefs[s.key]) })),
        );
        this.loadingNotifications.set(false);
        this.notificationPreferencesLoaded = true;
      },
      error: (err: unknown) => {
        this.notificationsError.set((err as { message?: string }).message || 'Erro desconhecido');
        this.loadingNotifications.set(false);
      },
    });
  }

  toggleNotification(key: NotificationPreferenceKey, event: Event): void {
    const input = event.target as HTMLInputElement;
    const newValue = input.checked;
    this.savingNotifications.set(true);
    this.notificationsApi.updatePreference(key, newValue).subscribe({
      next: () => {
        this.notificationSettings.update((settings) =>
          settings.map((s) => (s.key === key ? { ...s, enabled: signal(newValue) } : s)),
        );
        this.savingNotifications.set(false);
      },
      error: (err: unknown) => {
        this.savingNotifications.set(false);
        this.notificationsError.set((err as { message?: string }).message || 'Erro desconhecido');

        this.notificationSettings.update((settings) =>
          settings.map((s) => (s.key === key ? { ...s, enabled: signal(!newValue) } : s)),
        );
      },
    });
  }
}
