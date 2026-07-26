import React from 'react';

interface HeaderProps {
  title: string;
}

const Header: React.FC<HeaderProps> = ({ title }) => {
  return (
    <header className="app-header">
      <h1 className="app-title">{title}</h1>
    </header>
  );
};

export default Header;