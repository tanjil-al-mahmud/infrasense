import { useAuth } from '../contexts/AuthContext';

export const usePermissions = () => {
  const { user } = useAuth();
  return {
    isAdmin: user?.role === 'admin',
    isOperator: user?.role === 'operator' || user?.role === 'admin',
    isViewer: true,
    canManageUsers: user?.role === 'admin',
    canManageDevices: user?.role === 'admin' || user?.role === 'operator',
    canAcknowledgeAlerts: user?.role === 'admin' || user?.role === 'operator',
    canChangeSettings: user?.role === 'admin',
  };
};
