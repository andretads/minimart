import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { ProfileComponent } from './profile.component';
import { CustomerApiService } from '../customer-api.service';
import { AvatarApiService } from '../avatar-api.service';
import { NotificationsApiService } from '@features/notifications/notifications-api.service';
import { AuthService } from '@core/auth/auth.service';
import { AuthStore } from '@core/auth/auth.store';
import { ToastService } from '@core/toast/toast.service';
import { Address } from '@shared/models/address.models';

const customerResponse = {
  id: 'c1',
  userId: 'u1',
  displayName: 'Ana',
  status: 'active',
  version: 1,
  createdAt: 't',
  updatedAt: 't',
};

const address = (over: Partial<Address> = {}): Address => ({
  id: 'a1',
  customerId: 'c1',
  label: 'Casa',
  fullName: 'Ana',
  street: 'Rua 1',
  city: 'SP',
  state: 'SP',
  zip: '00000',
  country: 'BR',
  phone: '',
  isDefault: true,
  version: 1,
  createdAt: 't',
  updatedAt: 't',
  ...over,
});

describe('ProfileComponent', () => {
  let api: {
    getByUserId: ReturnType<typeof vi.fn>;
    update: ReturnType<typeof vi.fn>;
    getAddresses: ReturnType<typeof vi.fn>;
    createAddress: ReturnType<typeof vi.fn>;
    updateAddress: ReturnType<typeof vi.fn>;
    deleteAddress: ReturnType<typeof vi.fn>;
    setDefaultAddress: ReturnType<typeof vi.fn>;
  };
  let avatarApi: { get: ReturnType<typeof vi.fn>; upload: ReturnType<typeof vi.fn> };
  let notificationsApi: {
    getPreferences: ReturnType<typeof vi.fn>;
    updatePreference: ReturnType<typeof vi.fn>;
  };
  let authService: { me: ReturnType<typeof vi.fn>; changePassword: ReturnType<typeof vi.fn> };
  let authStore: { userId: ReturnType<typeof vi.fn>; user: ReturnType<typeof vi.fn> };
  let toast: { success: ReturnType<typeof vi.fn>; error: ReturnType<typeof vi.fn> };

  beforeEach(async () => {
    api = {
      getByUserId: vi.fn(() => of(customerResponse)),
      update: vi.fn(() => of({ ...customerResponse, displayName: 'Novo', version: 2 })),
      getAddresses: vi.fn(() => of([address()])),
      createAddress: vi.fn(() => of(address({ id: 'a2' }))),
      updateAddress: vi.fn(() => of(address())),
      deleteAddress: vi.fn(() => of(void 0)),
      setDefaultAddress: vi.fn(() => of(void 0)),
    };
    avatarApi = {
      get: vi.fn(() => of({ imageUrl: 'http://img' })),
      upload: vi.fn(() => of({ imageUrl: 'http://new' })),
    };
    notificationsApi = {
      getPreferences: vi.fn(() =>
        of({ emailEnabled: true, smsEnabled: false, pushEnabled: true, marketingEnabled: true }),
      ),
      updatePreference: vi.fn(() => of(void 0)),
    };
    authService = {
      me: vi.fn(() => of({ email: 'a@b.com', username: 'ana' })),
      changePassword: vi.fn(() => of(void 0)),
    };
    authStore = { userId: vi.fn(() => 'u1'), user: vi.fn(() => ({ username: 'ana' })) };
    toast = { success: vi.fn(), error: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [ProfileComponent],
      providers: [
        { provide: CustomerApiService, useValue: api },
        { provide: AvatarApiService, useValue: avatarApi },
        { provide: NotificationsApiService, useValue: notificationsApi },
        { provide: AuthService, useValue: authService },
        { provide: AuthStore, useValue: authStore },
        { provide: ToastService, useValue: toast },
      ],
    }).compileComponents();
  });

  it('loads profile, avatar, me and addresses on init', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    expect(api.getByUserId).toHaveBeenCalled();
    expect(authService.me).toHaveBeenCalled();
    expect(avatarApi.get).toHaveBeenCalledWith('u1');
    expect(f.componentInstance.customer()?.displayName).toBe('Ana');
    expect(f.componentInstance.avatarUrl()).toBe('http://img');
    expect(f.componentInstance.loading()).toBe(false);
    expect(api.getAddresses).toHaveBeenCalled();
  });

  it('falls back to auth store username when me() fails', () => {
    authService.me.mockReturnValue(throwError(() => new Error('x')));
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    expect(f.componentInstance.form.value.username).toBe('ana');
  });

  it('sets error when customer load fails', () => {
    api.getByUserId.mockReturnValue(throwError(() => ({ message: 'denied' })));
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    expect(f.componentInstance.error()).toBe('denied');
    expect(f.componentInstance.loading()).toBe(false);
  });

  it('restricted reflects 403 in error', () => {
    api.getByUserId.mockReturnValue(throwError(() => ({ message: '403 Forbidden' })));
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    expect(f.componentInstance.restricted()).toBe(true);
  });

  it('loadAvatar ignores 404 silently and clears url', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    avatarApi.get.mockReturnValue(throwError(() => ({ status: 404, message: 'no' })));
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    expect(f.componentInstance.avatarUrl()).toBeNull();
    expect(warn.mock.calls.some((c) => String(c[0]).includes('Failed to load avatar'))).toBe(false);
    warn.mockRestore();
  });

  it('loadAvatar warns on non-404', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    avatarApi.get.mockReturnValue(throwError(() => ({ status: 500, message: 'boom' })));
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });

  it('setTab loads notification preferences once', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.setTab('notifications');
    expect(notificationsApi.getPreferences).toHaveBeenCalledTimes(1);
    f.componentInstance.setTab('notifications');
    expect(notificationsApi.getPreferences).toHaveBeenCalledTimes(1);
  });

  it('save updates customer and shows success', () => {
    vi.useFakeTimers();
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.form.patchValue({ displayName: 'Novo' });
    f.componentInstance.save();
    expect(api.update).toHaveBeenCalledWith('c1', { displayName: 'Novo', version: 1 });
    expect(f.componentInstance.saveSuccess()).toBe(true);
    vi.advanceTimersByTime(3000);
    expect(f.componentInstance.saveSuccess()).toBe(false);
    vi.useRealTimers();
  });

  it('save no-ops when invalid or no customer', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.form.patchValue({ displayName: '' });
    f.componentInstance.save();
    expect(api.update).not.toHaveBeenCalled();
  });

  it('save handles 409 conflict', () => {
    api.update.mockReturnValue(throwError(() => ({ status: 409, message: 'conflict' })));
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.form.patchValue({ displayName: 'Novo' });
    f.componentInstance.save();
    expect(f.componentInstance.displayNameServerError()).toContain('outra sessão');
    expect(f.componentInstance.saving()).toBe(false);
  });

  it('save handles 400 field error', () => {
    api.update.mockReturnValue(
      throwError(() => ({ status: 400, fields: { displayName: ['ruim'] } })),
    );
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.form.patchValue({ displayName: 'Novo' });
    f.componentInstance.save();
    expect(f.componentInstance.displayNameServerError()).toBe('ruim');
  });

  it('save toasts generic error', () => {
    api.update.mockReturnValue(throwError(() => ({ status: 500, message: 'oops' })));
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.form.patchValue({ displayName: 'Novo' });
    f.componentInstance.save();
    expect(toast.error).toHaveBeenCalledWith('oops');
  });

  it('resetForm restores values', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.form.patchValue({ displayName: 'Changed', version: 5 });
    f.componentInstance.resetForm();
    expect(f.componentInstance.form.value.displayName).toBe('Ana');
    expect(f.componentInstance.form.value.version).toBe(1);
  });

  it('displayNameError reports required', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.componentInstance.form.get('displayName')!.setValue('');
    f.componentInstance.form.get('displayName')!.markAsTouched();
    expect(f.componentInstance.displayNameError()).toBe('Nome de exibição é obrigatório');
  });

  it('changePassword succeeds and resets form', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.passwordForm.setValue({
      currentPassword: 'old123',
      newPassword: 'new123',
      confirmNewPassword: 'new123',
    });
    f.componentInstance.changePassword();
    expect(authService.changePassword).toHaveBeenCalledWith({
      currentPassword: 'old123',
      newPassword: 'new123',
      confirmNewPassword: 'new123',
    });
    expect(toast.success).toHaveBeenCalled();
  });

  it('changePassword maps incorrect current password', () => {
    authService.changePassword.mockReturnValue(
      throwError(() => ({ status: 400, message: 'senha atual incorreta' })),
    );
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.passwordForm.setValue({
      currentPassword: 'bad123',
      newPassword: 'new123',
      confirmNewPassword: 'new123',
    });
    f.componentInstance.changePassword();
    expect(f.componentInstance.currentPasswordServerError()).toBe('Senha atual incorreta');
  });

  it('password error computeds', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.passwordForm.get('currentPassword')!.markAsTouched();
    f.componentInstance.passwordForm.get('newPassword')!.markAsTouched();
    f.componentInstance.passwordForm.get('confirmNewPassword')!.markAsTouched();
    expect(f.componentInstance.currentPasswordError()).toBe('Senha atual é obrigatória');
    expect(f.componentInstance.newPasswordError()).toBe('Nova senha é obrigatória');
    expect(f.componentInstance.confirmNewPasswordError()).toBe('Confirme sua senha');
  });

  it('addAddress opens editing with empty form', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.addAddress();
    expect(f.componentInstance.editing()).toBe(true);
    expect(f.componentInstance.editingAddressId()).toBeNull();
    expect(f.componentInstance.addressForm.value.country).toBe('BR');
  });

  it('editAddress populates form', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.editAddress(address());
    expect(f.componentInstance.editing()).toBe(true);
    expect(f.componentInstance.editingAddressId()).toBe('a1');
    expect(f.componentInstance.addressForm.value.label).toBe('Casa');
  });

  it('cancelEdit resets editing state', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.editAddress(address());
    f.componentInstance.cancelEdit();
    expect(f.componentInstance.editing()).toBe(false);
    expect(f.componentInstance.editingAddressId()).toBeNull();
  });

  it('saveAddress creates when not editing', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.addAddress();
    f.componentInstance.addressForm.patchValue({
      label: 'Casa',
      fullName: 'Ana',
      street: 'R',
      city: 'SP',
      state: 'SP',
      zip: '0',
      country: 'BR',
    });
    f.componentInstance.saveAddress();
    expect(api.createAddress).toHaveBeenCalled();
    expect(toast.success).toHaveBeenCalled();
  });

  it('saveAddress updates when editing', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.editAddress(address());
    f.componentInstance.saveAddress();
    expect(api.updateAddress).toHaveBeenCalledWith('a1', expect.objectContaining({ version: 1 }));
  });

  it('saveAddress maps 400 field errors', () => {
    api.createAddress.mockReturnValue(
      throwError(() => ({ status: 400, fields: { label: ['bad'] } })),
    );
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.addAddress();
    f.componentInstance.addressForm.patchValue({
      label: 'Casa',
      fullName: 'Ana',
      street: 'R',
      city: 'SP',
      state: 'SP',
      zip: '0',
      country: 'BR',
    });
    f.componentInstance.saveAddress();
    expect(f.componentInstance.getFirstError('label')).toBe('bad');
  });

  it('saveAddress handles 409 conflict', () => {
    api.createAddress.mockReturnValue(throwError(() => ({ status: 409 })));
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.addAddress();
    f.componentInstance.addressForm.patchValue({
      label: 'Casa',
      fullName: 'Ana',
      street: 'R',
      city: 'SP',
      state: 'SP',
      zip: '0',
      country: 'BR',
    });
    f.componentInstance.saveAddress();
    expect(toast.error).toHaveBeenCalledWith(expect.stringContaining('outra sessão'));
  });

  it('setDefaultAddress skips when already default', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.setDefaultAddress(address({ isDefault: true }));
    expect(api.setDefaultAddress).not.toHaveBeenCalled();
  });

  it('setDefaultAddress updates when not default', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.setDefaultAddress(address({ isDefault: false }));
    expect(api.setDefaultAddress).toHaveBeenCalledWith('a1');
    expect(toast.success).toHaveBeenCalled();
  });

  it('requestDeleteAddress then deleteAddress', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.requestDeleteAddress(address());
    expect(f.componentInstance.confirmDeleteAddress()).toBe(true);
    f.componentInstance.deleteAddress();
    expect(api.deleteAddress).toHaveBeenCalledWith('a1');
    expect(toast.success).toHaveBeenCalled();
    expect(f.componentInstance.confirmDeleteAddress()).toBe(false);
  });

  it('deleteAddress no-ops without pending address', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.deleteAddress();
    expect(api.deleteAddress).not.toHaveBeenCalled();
  });

  it('toggleNotification updates on success', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.setTab('notifications');
    const event = { target: { checked: false } } as unknown as Event;
    f.componentInstance.toggleNotification('smsEnabled', event);
    expect(notificationsApi.updatePreference).toHaveBeenCalledWith('smsEnabled', false);
  });

  it('toggleNotification reverts on failure', () => {
    notificationsApi.updatePreference.mockReturnValue(throwError(() => ({ message: 'no' })));
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.setTab('notifications');
    const event = { target: { checked: false } } as unknown as Event;
    f.componentInstance.toggleNotification('smsEnabled', event);
    expect(f.componentInstance.notificationsError()).toBe('no');
  });

  it('loadNotificationPreferences sets error on failure', () => {
    notificationsApi.getPreferences.mockReturnValue(throwError(() => ({ message: 'pref-fail' })));
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.setTab('notifications');
    expect(f.componentInstance.notificationsError()).toBe('pref-fail');
  });

  it('loadAddresses sets error on failure', () => {
    api.getAddresses.mockReturnValue(throwError(() => ({ message: 'addr-fail' })));
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    expect(f.componentInstance.addressesError()).toBe('addr-fail');
  });

  it('onAvatarSelected rejects invalid type', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    const file = new File(['x'], 'a.txt', { type: 'text/plain' });
    const event = { target: { files: [file], value: 'a.txt' } } as unknown as Event;
    f.componentInstance.onAvatarSelected(event);
    expect(toast.error).toHaveBeenCalledWith(expect.stringContaining('Formato'));
  });

  it('onAvatarSelected rejects oversized file', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    const file = new File(['x'], 'a.png', { type: 'image/png' });
    Object.defineProperty(file, 'size', { value: 6 * 1024 * 1024 });
    const event = { target: { files: [file], value: 'a.png' } } as unknown as Event;
    f.componentInstance.onAvatarSelected(event);
    expect(toast.error).toHaveBeenCalledWith(expect.stringContaining('grande'));
  });

  it('onAvatarSelected no-ops without file', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    const event = { target: { files: [], value: '' } } as unknown as Event;
    f.componentInstance.onAvatarSelected(event);
    expect(avatarApi.upload).not.toHaveBeenCalled();
  });

  it('onAvatarSelected uploads valid file', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    const file = new File(['x'], 'a.png', { type: 'image/png' });
    const event = { target: { files: [file], value: 'a.png' } } as unknown as Event;
    f.componentInstance.onAvatarSelected(event);
    expect(avatarApi.upload).toHaveBeenCalledWith(file, 'u1');
    expect(f.componentInstance.avatarUrl()).toBe('http://new');
    expect(toast.success).toHaveBeenCalled();
  });

  it('onAvatarImageError clears avatar url', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    f.componentInstance.onAvatarImageError();
    expect(f.componentInstance.avatarUrl()).toBeNull();
  });

  it('formatDate renders pt-BR date', () => {
    const f = TestBed.createComponent(ProfileComponent);
    f.detectChanges();
    expect(f.componentInstance.formatDate('2026-05-01T00:00:00Z')).toContain('2026');
  });
});
