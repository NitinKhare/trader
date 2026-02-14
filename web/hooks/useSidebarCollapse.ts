"use client";

import { useEffect, useRef, useState } from "react";

const SIDEBAR_STORAGE_KEY = "trading-dashboard-sidebar-collapsed";

export function useSidebarCollapse() {
  const [isCollapsed, setIsCollapsed] = useState(false);
  const isInitializedRef = useRef(false);

  // Load from localStorage on mount (only once)
  useEffect(() => {
    if (!isInitializedRef.current) {
      try {
        const saved = localStorage.getItem(SIDEBAR_STORAGE_KEY);
        if (saved !== null) {
          setIsCollapsed(JSON.parse(saved));
        }
      } catch (e) {
        console.error("Failed to load sidebar state:", e);
      }
      isInitializedRef.current = true;
    }
  }, []);

  // Save to localStorage when changed
  useEffect(() => {
    if (isInitializedRef.current) {
      try {
        localStorage.setItem(SIDEBAR_STORAGE_KEY, JSON.stringify(isCollapsed));
      } catch (e) {
        console.error("Failed to save sidebar state:", e);
      }
    }
  }, [isCollapsed]);

  const toggleSidebar = () => {
    setIsCollapsed((prev) => !prev);
  };

  return {
    isCollapsed,
    toggleSidebar,
  };
}
