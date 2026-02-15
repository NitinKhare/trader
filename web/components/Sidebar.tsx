"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useSidebarCollapse } from "@/hooks/useSidebarCollapse";

const menuItems = [
  { icon: "📊", label: "Dashboard", href: "/dashboard" },
  { icon: "📈", label: "Stocks", href: "/stocks" },
  { icon: "🔬", label: "Backtest", href: "/backtest" },
  { icon: "📚", label: "Documentation", href: "/docs" },
  { icon: "🏠", label: "Home", href: "/" },
];

export default function Sidebar() {
  const pathname = usePathname();
  const { isCollapsed, toggleSidebar } = useSidebarCollapse();

  return (
    <>
      {/* Sidebar */}
      <aside
        className={`fixed left-0 top-0 h-screen bg-white border-r border-slate-200 shadow-lg flex flex-col transition-all duration-300 ease-in-out z-50 ${
          isCollapsed ? "w-20" : "w-64"
        }`}
      >
        {/* Logo Section */}
        <div
          className={`flex items-center gap-3 p-4 border-b border-slate-200 transition-all duration-300 ${
            isCollapsed ? "justify-center" : "justify-start"
          }`}
        >
          {/* Logo Icon */}
          <div className="w-10 h-10 bg-gradient-to-br from-blue-600 via-blue-700 to-indigo-700 rounded-lg flex items-center justify-center text-white font-black text-lg flex-shrink-0 shadow-md">
            ₹
          </div>

          {/* Logo Text - Hidden when collapsed */}
          {!isCollapsed && (
            <div className="flex flex-col min-w-0">
              <h1 className="text-lg font-black text-slate-900 truncate">
                Trading
              </h1>
              <p className="text-xs text-slate-500 font-semibold truncate">
                Dashboard
              </p>
            </div>
          )}
        </div>

        {/* Collapse Toggle Button */}
        <div className="px-2 py-3 border-b border-slate-200">
          <button
            onClick={toggleSidebar}
            className="w-full h-10 flex items-center justify-center rounded-lg hover:bg-slate-100 transition-colors duration-200 text-slate-600 hover:text-slate-900"
            title={isCollapsed ? "Expand" : "Collapse"}
          >
            <span className={`text-xl transition-transform duration-300 ${isCollapsed ? "" : "rotate-180"}`}>
              {isCollapsed ? "➡️" : "⬅️"}
            </span>
          </button>
        </div>

        {/* Navigation Menu */}
        <nav className="flex-1 p-3 space-y-2 overflow-y-auto">
          {menuItems.map((item) => {
            const isActive = pathname === item.href;
            return (
              <div key={item.href} className="group relative">
                <Link
                  href={item.href}
                  className={`flex items-center gap-3 px-3 py-3 rounded-xl font-semibold transition-all duration-200 ${
                    isActive
                      ? "bg-blue-50 text-blue-700 shadow-md"
                      : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
                  } ${isCollapsed ? "justify-center" : "justify-start"}`}
                  title={item.label}
                >
                  <span className="text-xl flex-shrink-0">{item.icon}</span>
                  {!isCollapsed && <span className="truncate">{item.label}</span>}
                </Link>

                {/* Tooltip for collapsed state */}
                {isCollapsed && (
                  <div className="absolute left-full top-1/2 -translate-y-1/2 ml-2 px-3 py-2 bg-slate-900 text-white text-sm font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-200 whitespace-nowrap pointer-events-none z-50">
                    {item.label}
                    <div className="absolute right-full top-1/2 -translate-y-1/2 border-4 border-transparent border-r-slate-900"></div>
                  </div>
                )}
              </div>
            );
          })}
        </nav>

        {/* Status Section */}
        <div
          className={`border-t border-slate-200 p-3 space-y-2 transition-all duration-300 ${
            isCollapsed ? "text-center" : "text-left"
          }`}
        >
          {!isCollapsed && (
            <p className="text-xs font-bold text-slate-600 uppercase tracking-widest">
              Status
            </p>
          )}

          {/* Connection Badge */}
          <div
            className={`flex items-center gap-2 px-3 py-2 rounded-lg bg-emerald-50 border border-emerald-200 transition-all duration-200 ${
              isCollapsed ? "justify-center" : "justify-start"
            }`}
            title="Connected"
          >
            <div className="w-2.5 h-2.5 bg-emerald-500 rounded-full animate-pulse flex-shrink-0"></div>
            {!isCollapsed && (
              <span className="text-xs font-semibold text-emerald-700 truncate">
                Connected
              </span>
            )}
          </div>

          {!isCollapsed && (
            <p className="text-xs text-slate-400 px-3">Backend: :8081</p>
          )}
        </div>
      </aside>

      {/* Spacer div for fixed sidebar - maintains layout flow */}
      <div
        className={`flex-shrink-0 transition-all duration-300 ${
          isCollapsed ? "w-20" : "w-64"
        }`}
      ></div>
    </>
  );
}
