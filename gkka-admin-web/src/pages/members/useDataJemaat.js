import { useEffect, useRef, useState } from 'react';
import apiClient from '../../services/api';
import { getSession } from '../../services/session';
import { createForm } from './jemaatFields';

export default function useDataJemaat() {
  const [jemaat, setJemaat] = useState([]);
  const [meta, setMeta] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState('');
  const [notice, setNotice] = useState('');

  const [searchQuery, setSearchQuery] = useState('');
  const [filters, setFilters] = useState({
    q: '',
    role: '',
    status_verifikasi: '',
  });
  const [page, setPage] = useState(1);
  const [revision, setRevision] = useState(0);

  const [selectedUser, setSelectedUser] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const selectionVersion = useRef(0);

  // null | create | edit | role | password
  const [modal, setModal] = useState(null);
  const [formData, setFormData] = useState(createForm);
  const [formError, setFormError] = useState('');
  const [fieldErrors, setFieldErrors] = useState({});
  const [isSaving, setIsSaving] = useState(false);
  const [isBusy, setIsBusy] = useState(false);
  const mutationLock = useRef(false);

  const currentUserID = getSession()?.user?.id;
  const isSelf = selectedUser?.id === currentUserID;

  useEffect(() => {
    const controller = new AbortController();

    async function load() {
      setIsLoading(true);
      setErrorMsg('');

      try {
        const response = await apiClient.get('/users', {
          params: { ...filters, page, limit: 20 },
          signal: controller.signal,
        });

        if (controller.signal.aborted) return;
        setJemaat(response.data.data.items || []);
        setMeta(response.data.data.meta);
      } catch (error) {
        if (controller.signal.aborted) return;

        // Setelah penghapusan, halaman terakhir mungkin sudah tidak ada.
        if (error.response?.status === 404 && page > 1) {
          setPage(value => Math.max(1, value - 1));
          return;
        }

        setErrorMsg(
          error.response?.data?.message || 'Gagal memuat data jemaat.',
        );
        setJemaat([]);
        setMeta(null);
      } finally {
        if (!controller.signal.aborted) setIsLoading(false);
      }
    }

    load();
    return () => controller.abort();
  }, [filters, page, revision]);

  function refreshList() {
    setRevision(value => value + 1);
  }

  function closeDrawer() {
    selectionVersion.current += 1;
    setSelectedUser(null);
    setDetailLoading(false);
  }

  function applyFilter(key, value) {
    closeDrawer();
    setPage(1);
    setFilters(previous => ({ ...previous, [key]: value }));
  }

  function submitSearch(event) {
    event.preventDefault();
    applyFilter('q', searchQuery.trim());
  }

  async function openDetail(user) {
    const version = ++selectionVersion.current;
    setSelectedUser(null);
    setDetailLoading(true);
    setErrorMsg('');

    try {
      const response = await apiClient.get(`/users/${user.id}`);
      if (version !== selectionVersion.current) return;
      setSelectedUser(response.data.data);
    } catch (error) {
      if (version !== selectionVersion.current) return;
      setErrorMsg(
        error.response?.data?.message || 'Gagal mengambil detail jemaat.',
      );
    } finally {
      if (version === selectionVersion.current) setDetailLoading(false);
    }
  }

  function openModal(mode) {
    setFormError('');
    setFieldErrors({});
    setNotice('');
    setFormData(createForm(mode === 'create' ? null : selectedUser));
    setModal(mode);
  }

  function closeModal() {
    if (!isSaving) setModal(null);
  }

  function changeField(key, value) {
    setFormData(previous => ({ ...previous, [key]: value }));
  }

  function changeIdentity(key, value) {
    setFormData(previous => ({
      ...previous,
      identitas: { ...previous.identitas, [key]: value },
    }));
  }

  async function submitForm(event) {
    event.preventDefault();
    if (mutationLock.current) return;

    const targetID = selectedUser?.id;
    if (modal !== 'create' && !targetID) return;

    if (
      (modal === 'password' || modal === 'create') &&
      new TextEncoder().encode(formData.password).length > 72
    ) {
      setFormError('Password maksimal 72 byte.');
      return;
    }

    mutationLock.current = true;
    setIsSaving(true);
    setFormError('');
    setFieldErrors({});

    try {
      if (modal === 'create' || modal === 'edit') {
        const payload = {
          kode_jemaat: formData.kode_jemaat.trim(),
          nama: formData.nama.trim(),
          email: formData.email.trim(),
          no_telepon: formData.no_telepon.trim(),
          identitas: { ...formData.identitas },
        };

        if (modal === 'create') {
          const response = await apiClient.post('/users', {
            ...payload,
            role: formData.role,
            password: formData.password,
          });
          setSelectedUser(response.data.data);
          setPage(1);
        } else {
          const response = await apiClient.put(`/users/${targetID}`, payload);
          setSelectedUser(response.data.data);
        }
      } else if (modal === 'role') {
        const response = await apiClient.patch(`/users/${targetID}/role`, {
          role: formData.role,
        });
        setSelectedUser(response.data.data);
      } else if (modal === 'password') {
        await apiClient.put(`/users/${targetID}/password`, {
          password: formData.password,
        });
        setSelectedUser(previous =>
          previous?.id === targetID
            ? { ...previous, account_activated: true }
            : previous,
        );
      }

      setModal(null);
      setNotice('Perubahan berhasil disimpan.');
      refreshList();
    } catch (error) {
      setFormError(
        error.response?.data?.message || 'Gagal menyimpan perubahan.',
      );
      setFieldErrors(error.response?.data?.errors || {});
    } finally {
      mutationLock.current = false;
      setIsSaving(false);
    }
  }

  async function toggleActive() {
    if (!selectedUser || mutationLock.current) return;
    if (isSelf && selectedUser.is_aktif) return;

    const target = selectedUser;
    const label = target.is_aktif ? 'menonaktifkan' : 'mengaktifkan';
    if (!window.confirm(`Yakin ${label} akun ${target.nama}?`)) return;

    mutationLock.current = true;
    setIsBusy(true);
    setErrorMsg('');
    setNotice('');

    try {
      const response = await apiClient.patch(`/users/${target.id}/status`, {
        is_aktif: !target.is_aktif,
      });
      setSelectedUser(response.data.data);
      setNotice('Status akun berhasil diperbarui.');
      refreshList();
    } catch (error) {
      setErrorMsg(
        error.response?.data?.message || 'Gagal mengubah status akun.',
      );
    } finally {
      mutationLock.current = false;
      setIsBusy(false);
    }
  }

  async function deleteUser() {
    if (!selectedUser || isSelf || mutationLock.current) return;
    const target = selectedUser;

    if (!window.confirm(`Hapus jemaat "${target.nama}"?`)) return;

    mutationLock.current = true;
    setIsBusy(true);
    setErrorMsg('');
    setNotice('');

    try {
      await apiClient.delete(`/users/${target.id}`);
      closeDrawer();
      setNotice('Jemaat berhasil dihapus.');
      refreshList();
    } catch (error) {
      setErrorMsg(
        error.response?.data?.message || 'Gagal menghapus jemaat.',
      );
    } finally {
      mutationLock.current = false;
      setIsBusy(false);
    }
  }

  return {
    jemaat, meta, isLoading, errorMsg, notice,
    searchQuery, setSearchQuery, filters, applyFilter, submitSearch,
    page, setPage,
    selectedUser, detailLoading, openDetail, closeDrawer,
    modal, openModal, closeModal, formData, changeField, changeIdentity,
    formError, fieldErrors, isSaving, submitForm,
    isBusy, isSelf, toggleActive, deleteUser,
  };
}