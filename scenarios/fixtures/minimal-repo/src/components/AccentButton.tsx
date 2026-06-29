import React, { useState } from 'react';
import { useTheme } from '../ThemeContext';

interface AccentButtonProps {
  children: React.ReactNode;
}

const AccentButton: React.FC<AccentButtonProps> = ({ children }) => {
  const theme = useTheme();
  
  return (
    <button 
      className="btn-primary" 
      style={{ 
        padding: '10px 20px', 
        cursor: 'pointer', 
        transition: 'all 0.3s ease',
        backgroundColor: theme.accentColor,
        color: theme.textColor
      }}
    >
      {children}
    </button>
  );
};

export default AccentButton;