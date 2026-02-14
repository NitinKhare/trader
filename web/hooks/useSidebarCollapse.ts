"use client";

import { useState, useEffect } from "react";

const SIDEBAR_STORAGE_KEY = "trading-dashboard-sidebar-collapsed";

export function useSidebarCollapse() {
  const [isCollapsed, setIsCollapsed] = useState(false);
  const [isMounted, setIsMounted] = useState(false);

  // Load from localStorage on mount
  useEffect(() => {
    const saved = localStorage.getItem(SIDEBAR_STORAGE_KEY);
    if (saved) {
      setIsCollapsed(JSON.parse(saved));
    }
    setIsMounted(true);
  }, []);

  // Save to localStorage when changed
  useEffect(() => {
    if (isMounted) {
      localStorage.setItem(SIDEBAR_STORAGE_KEY, JSON.stringify(isCollapsed));
    }
  }, [isCollapsed, isMounted]);

  const toggleSidebar = () => {
    setIsCollapsed((prev) => !prev);
  };

  return {
    isCollapsed,
    toggleSidebar,
    isMounted,
  };
}
