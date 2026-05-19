import { Routes, Route, Navigate } from 'react-router-dom';
import { AppLayout } from './components/AppLayout';
import { Dashboard } from './pages/Dashboard';
import { Timeline } from './pages/Timeline';
import { Categories } from './pages/Categories';
import { Coding } from './pages/Coding';
import { Gaming } from './pages/Gaming';
import { Devices } from './pages/Devices';
import { Shortcuts } from './pages/Shortcuts';
import { Settings } from './pages/Settings';
import { DeviceProvider } from './state/deviceContext';

export default function App() {
  return (
    <DeviceProvider>
      <AppLayout>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/coding" element={<Coding />} />
          <Route path="/gaming" element={<Gaming />} />
          <Route path="/timeline" element={<Timeline />} />
          <Route path="/categories" element={<Categories />} />
          <Route path="/shortcuts" element={<Shortcuts />} />
          <Route path="/devices" element={<Devices />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AppLayout>
    </DeviceProvider>
  );
}
