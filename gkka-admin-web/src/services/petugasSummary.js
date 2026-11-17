import api from './api';

export async function getPetugasSummary(activities, signal) {
  const result = {};
  let nextIndex = 0;

  async function worker() {
    while (nextIndex < activities.length) {
      if (signal?.aborted) return;

      const activity = activities[nextIndex];
      nextIndex += 1;

      const response = await api.get(
        `/kegiatan/${activity.id}/petugas`,
        { signal },
      );

      const assignments = response.data.data.items || [];

      const activeAssignments = assignments.filter(
        item => item.status_konfirmasi !== 'DITOLAK',
      );

      result[activity.id] = {
        people: new Set(
          activeAssignments.map(item => item.id_user),
        ).size,
        assignments: activeAssignments.length,
      };
    }
  }

  await Promise.all(
    Array.from(
      { length: Math.min(4, activities.length) },
      () => worker(),
    ),
  );

  return result;
}