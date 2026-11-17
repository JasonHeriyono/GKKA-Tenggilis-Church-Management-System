import { useEffect, useRef, useState } from 'react';
import api from '../../services/api';

function today() {
  const date = new Date();
  const pad = value => String(value).padStart(2, '0');

  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

function message(error) {
  const body = error.response?.data;

  return [
    body?.message || 'Tidak dapat memproses permintaan.',
    ...Object.entries(body?.errors || {}).map(
      ([field, value]) => `${field}: ${value}`,
    ),
  ].join('\n');
}

async function getAllPages(url, params, signal) {
  const items = [];
  let page = 1;

  while (true) {
    const response = await api.get(url, {
      params: { ...params, page, limit: 100 },
      signal,
    });

    const data = response.data.data;
    items.push(...(data.items || []));

    if (page >= (data.meta?.total_pages || 1)) break;
    page += 1;
  }

  return items;
}

export default function usePenugasanPelayan() {
  const initialParams = useRef(
    new URLSearchParams(window.location.search),
  );

  const [date, setDate] = useState(() => {
    const requested = initialParams.current.get('tanggal');

    if (requested && /^\d{4}-\d{2}-\d{2}$/.test(requested)) {
      const parsed = new Date(`${requested}T12:00:00`);

      if (!Number.isNaN(parsed.getTime())) {
        const formatted =
          `${parsed.getFullYear()}-` +
          `${String(parsed.getMonth() + 1).padStart(2, '0')}-` +
          `${String(parsed.getDate()).padStart(2, '0')}`;

        if (formatted === requested) return requested;
      }
    }

    return today();
  });
  const [activities, setActivities] = useState([]);
  const [activityID, setActivityID] = useState('');
  const [skills, setSkills] = useState([]);
  const [skillID, setSkillID] = useState('');
  const [assignments, setAssignments] = useState([]);
  const [candidates, setCandidates] = useState([]);
  const [selectedIDs, setSelectedIDs] = useState([]);
  const [search, setSearch] = useState('');

  const [loadingActivities, setLoadingActivities] = useState(true);
  const [loadingSkills, setLoadingSkills] = useState(true);
  const [loadingAssignments, setLoadingAssignments] = useState(false);
  const [loadingCandidates, setLoadingCandidates] = useState(false);

  const [error, setError] = useState('');
  const [candidateError, setCandidateError] = useState('');
  const [notice, setNotice] = useState('');
  const [warnings, setWarnings] = useState([]);
  const [busy, setBusy] = useState(false);
  const [progress, setProgress] = useState(null);
  const [revision, setRevision] = useState(0);
  const lock = useRef(false);

  useEffect(() => {
    const controller = new AbortController();

    async function load() {
      setLoadingActivities(true);
      setActivities([]);
      setActivityID('');
      setAssignments([]);
      setSelectedIDs([]);
      setWarnings([]);
      setError('');

      try {
        const items = await getAllPages(
          '/kegiatan',
          { dari: date, sampai: date },
          controller.signal,
        );

        if (controller.signal.aborted) return;

        setActivities(items);

        // MULAI PERUBAHAN PEMILIHAN KEGIATAN
        const requestedID = initialParams.current.get('kegiatan');

        const preferred = items.find(
          item => String(item.id) === requestedID,
        );

        setActivityID(
          preferred
            ? String(preferred.id)
            : items[0]
              ? String(items[0].id)
              : '',
        );

        initialParams.current.delete('kegiatan');
      } catch (err) {
        if (!controller.signal.aborted) setError(message(err));
      } finally {
        if (!controller.signal.aborted) setLoadingSkills(false);
      }
    }

        load();
    return () => controller.abort();
  }, [date, revision]);

  useEffect(() => {
    const controller = new AbortController();

    async function load() {
      setLoadingActivities(true);
      setActivities([]);
      setActivityID('');
      setAssignments([]);
      setSelectedIDs([]);
      setWarnings([]);
      setError('');

      try {
        const items = await getAllPages(
          '/kegiatan',
          { dari: date, sampai: date },
          controller.signal,
        );

        if (controller.signal.aborted) return;

        setActivities(items);
        setActivityID(items[0] ? String(items[0].id) : '');
      } catch (err) {
        if (!controller.signal.aborted) setError(message(err));
      } finally {
        if (!controller.signal.aborted) setLoadingActivities(false);
      }
    }

    load();
    return () => controller.abort();
  }, [date, revision]);

  useEffect(() => {
    if (!activityID) return;
    const controller = new AbortController();

    async function load() {
      setLoadingAssignments(true);
      setAssignments([]);

      try {
        const response = await api.get(`/kegiatan/${activityID}/petugas`, {
          signal: controller.signal,
        });

        if (!controller.signal.aborted) {
          setAssignments(response.data.data.items || []);
        }
      } catch (err) {
        if (!controller.signal.aborted) setError(message(err));
      } finally {
        if (!controller.signal.aborted) setLoadingAssignments(false);
      }
    }

    load();
    return () => controller.abort();
  }, [activityID]);

  useEffect(() => {
    if (!skillID) return;
    const controller = new AbortController();

    async function load() {
      setLoadingCandidates(true);
      setCandidates([]);
      setCandidateError('');

      try {
        const response = await api.get(`/keahlian/${skillID}`, {
          signal: controller.signal,
        });

        if (!controller.signal.aborted) {
          setCandidates(response.data.data.jemaat || []);
        }
      } catch (err) {
        if (!controller.signal.aborted) setCandidateError(message(err));
      } finally {
        if (!controller.signal.aborted) setLoadingCandidates(false);
      }
    }

    load();
    return () => controller.abort();
  }, [skillID, revision]);

  const activity = activities.find(item => String(item.id) === activityID);
  const skill = skills.find(item => String(item.id) === skillID);

  const assignedIDs = new Set(
    assignments
      .filter(item => String(item.id_keahlian) === skillID)
      .map(item => item.id_user),
  );

  const visibleCandidates = candidates.filter(user =>
    `${user.nama} ${user.kode_jemaat || ''}`
      .toLowerCase()
      .includes(search.trim().toLowerCase()),
  );

  const available = visibleCandidates.filter(
    user => !assignedIDs.has(user.id_user),
  );

  const allSelected =
    available.length > 0 &&
    available.every(user => selectedIDs.includes(user.id_user));

  function changeDate(value) {
    if (!value || lock.current) return;
    setDate(value);
    setNotice('');
  }

  function moveDate(offset) {
    const next = new Date(`${date}T12:00:00`);
    next.setDate(next.getDate() + offset);

    const pad = value => String(value).padStart(2, '0');
    changeDate(
      `${next.getFullYear()}-${pad(next.getMonth() + 1)}-${pad(next.getDate())}`,
    );
  }

  function chooseActivity(value) {
    if (lock.current) return;
    setActivityID(value);
    setAssignments([]);
    setSelectedIDs([]);
    setWarnings([]);
    setError('');
    setNotice('');
  }

  function chooseSkill(value) {
    if (lock.current) return;
    setSkillID(value);
    setCandidates([]);
    setSelectedIDs([]);
    setSearch('');
    setCandidateError('');
  }

  function toggleUser(userID) {
    if (lock.current) return;

    setSelectedIDs(previous =>
      previous.includes(userID)
        ? previous.filter(id => id !== userID)
        : [...previous, userID],
    );
  }

  function toggleAll() {
    if (lock.current) return;
    const ids = available.map(user => user.id_user);

    setSelectedIDs(previous =>
      allSelected
        ? previous.filter(id => !ids.includes(id))
        : [...new Set([...previous, ...ids])],
    );
  }

  async function assignSelected() {
    if (
      lock.current || !activityID || !skillID ||
      selectedIDs.length === 0 || loadingAssignments || loadingCandidates
    ) {
      return;
    }

    lock.current = true;
    setBusy(true);
    setError('');
    setNotice('');
    setWarnings([]);

    const ids = [...selectedIDs];
    const failures = [];
    const collectedWarnings = [];
    const resolved = new Set();
    let added = 0;
    let latest = [];
    let attempted = 0;

    const nameOf = id =>
      candidates.find(user => user.id_user === id)?.nama || `Jemaat ID ${id}`;

    async function refreshAssignments() {
      const response = await api.get(`/kegiatan/${activityID}/petugas`);
      latest = response.data.data.items || [];
      setAssignments(latest);
    }

    function exists(userID) {
      return latest.some(item =>
        item.id_user === userID &&
        String(item.id_keahlian) === skillID,
      );
    }

    function resolve(userID) {
      resolved.add(userID);
      setSelectedIDs(previous => previous.filter(id => id !== userID));
    }

    try {
      setProgress({ completed: 0, total: ids.length });
      await refreshAssignments();

      for (const userID of ids) {
        let stop = false;

        if (exists(userID)) {
          resolve(userID);
        } else {
          try {
            const response = await api.post(
              `/kegiatan/${activityID}/petugas`,
              { id_user: userID, id_keahlian: Number(skillID) },
            );

            const saved = response.data.data;
            latest = [...latest, saved];
            setAssignments(latest);
            added += 1;
            resolve(userID);

            for (const warning of saved.peringatan || []) {
              collectedWarnings.push(`${nameOf(userID)}: ${warning}`);
            }

            setWarnings([...collectedWarnings]);
          } catch (err) {
            const status = err.response?.status;
            let verified = false;

            if (!err.response || status === 409 || status >= 500) {
              try {
                await refreshAssignments();
                verified = exists(userID);
              } catch {
                // Hasil belum dapat dipastikan; pilihan dipertahankan.
              }
            }

            if (verified) {
              resolve(userID);
            } else {
              failures.push(`${nameOf(userID)}: ${message(err)}`);
            }

            stop =
              !err.response ||
              [401, 403, 404, 429].includes(status) ||
              status >= 500;
          }
        }

        attempted += 1;
        setProgress({ completed: attempted, total: ids.length });
        if (stop) break;
      }

      setNotice(
        `${added} penugasan baru tersimpan. ${resolved.size} dari ${ids.length} pilihan sudah terdaftar.`,
      );

      if (resolved.size < ids.length) {
        setError([
          `${ids.length - resolved.size} pilihan belum selesai dan tetap terpilih.`,
          ...failures,
        ].join('\n'));
      }
    } catch (err) {
      setError(`Belum dapat memeriksa penugasan terbaru.\n${message(err)}`);
    } finally {
      lock.current = false;
      setBusy(false);
      setProgress(null);
    }
  }

  async function removeAssignment(item) {
    if (lock.current) return;
    if (!window.confirm(`Batalkan penugasan ${item.nama} sebagai ${item.nama_keahlian}?`)) {
      return;
    }

    lock.current = true;
    setBusy(true);
    setError('');
    setNotice('');

    try {
      await api.delete(
        `/kegiatan/${activityID}/petugas/${item.id_penjadwalan}`,
      );

      setAssignments(previous =>
        previous.filter(row => row.id_penjadwalan !== item.id_penjadwalan),
      );
      setNotice('Penugasan berhasil dibatalkan.');
    } catch (err) {
      setError(message(err));
    } finally {
      lock.current = false;
      setBusy(false);
    }
  }

  return {
    date, changeDate, moveDate,
    activities, activityID, activity, chooseActivity,
    skills, skillID, skill, chooseSkill,
    assignments, visibleCandidates, available, assignedIDs,
    selectedIDs, setSelectedIDs, toggleUser, toggleAll, allSelected,
    search, setSearch,
    loadingActivities, loadingSkills, loadingAssignments, loadingCandidates,
    error, candidateError, notice, warnings, busy, progress,
    assignSelected, removeAssignment,
    reload: () => setRevision(value => value + 1),
  };
}