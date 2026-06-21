import React from 'react';

const Settings = () => {
  return (
    <div className="space-y-4">
      {/* Future Settings Items */}
      <div className="p-4 rounded-lg bg-neutral-100 dark:bg-neutral-800 text-center opacity-75">
        <p className="text-sm">Settings will appear here in future iterations.</p>
        <p className="text-xs mt-2">Current implementation: Minimal theme settings only.</p>
      </div>

      {/* Note for developers */}
      <div className="p-3 rounded-lg bg-neutral-100 dark:bg-neutral-800 text-left text-sm opacity-75">
        <p>Note:</p>
        <ul className="mt-2 list-disc list-inside space-y-1">
          <li>Future settings can be added here</li>
          <li>Connect to API/config files for persistence</li>
          <li>Add keyboard shortcuts section</li>
          <li>Include appearance preferences (accent color, etc.)</li>
        </ul>
      </div>
    </div>
  );
};

export default Settings;