import React from 'react';

interface LayoutProps {
  children: React.ReactNode;
  header?: React.ReactNode;
}

export const Layout: React.FC<LayoutProps> = ({ children, header }) => {
  return (
    <div className="h-full flex flex-col bg-white">
      {header}
      <div className="flex-1 flex flex-col overflow-hidden">
        {children}
      </div>
    </div>
  );
};