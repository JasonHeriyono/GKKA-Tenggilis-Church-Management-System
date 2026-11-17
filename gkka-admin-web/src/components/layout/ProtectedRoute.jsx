import { Navigate, Outlet } from 'react-router-dom';
import { getSession } from '../../services/session';
export default function ProtectedRoute({ allowedRoles }) {
  const session = getSession();
  if (!session || !['Super Admin', 'Majelis'].includes(session.user.role)) {
    return <Navigate to="/login" replace />;
  }
  if (allowedRoles && !allowedRoles.includes(session.user.role)) {
    return <Navigate to={session.user.role === 'Majelis' ? '/laporan-majelis' : '/'} replace />;
  }
  return <Outlet />;
}
