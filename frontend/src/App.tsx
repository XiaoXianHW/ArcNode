import { Routes, Route, Navigate } from 'react-router-dom';
import { AppLayout } from './components/AppLayout';
import { Dashboard } from './pages/Dashboard';
import { Timeline } from './pages/Timeline';
import { Categories } from './pages/Categories';
import { Coding } from './pages/Coding';
import { Gaming } from './pages/Gaming';
import { Insights } from './pages/Insights';
import { Focus } from './pages/Focus';
import { Wellness } from './pages/Wellness';
import { System } from './pages/System';
import { Profile } from './pages/Profile';
import { Devices } from './pages/Devices';
import { Shortcuts } from './pages/Shortcuts';
import { Settings } from './pages/Settings';
import { DeviceProvider } from './state/deviceContext';
import { ThemeProvider } from './state/themeContext';
import { I18nProvider } from './state/i18nContext';

export default function App() {
  return (
    <ThemeProvider>
      <I18nProvider>
        <DeviceProvider>
          <AppLayout>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/profile" element={<Profile />} />
              <Route path="/timeline" element={<Timeline />} />
              <Route path="/categories" element={<Categories />} />
              <Route path="/focus" element={<Focus />} />
              <Route path="/coding" element={<Coding />} />
              <Route path="/gaming" element={<Gaming />} />
              <Route path="/insights" element={<Insights />} />
              <Route path="/shortcuts" element={<Shortcuts />} />
              <Route path="/wellness" element={<Wellness />} />
              <Route path="/system" element={<System />} />
              <Route path="/devices" element={<Devices />} />
              <Route path="/settings" element={<Settings />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </AppLayout>
        </DeviceProvider>
      </I18nProvider>
    </ThemeProvider>
  );
}
