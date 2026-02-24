"use client";

import { useEffect, useState } from "react";
import { observer } from "mobx-react-lite";
import { BookOpen, Frame, Map } from "lucide-react";

import { NavMain } from "@/components/nav-main";
import { NavUser } from "@/components/nav-user";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar";

// This is sample data.
const data = {
  navMain: [
    {
      title: "Dashboard",
      url: "/dashboard",
      icon: Map,
      isActive: true,
    },
    {
      title: "Documentation",
      url: "https://docs.infralayer.dev",
      external: true,
      icon: BookOpen,
    },
    {
      title: "Integrations",
      url: "/integrations",
      icon: Frame,
    },
  ],
};

const ConsoleSidebar = observer(
  ({ ...props }: React.ComponentProps<typeof Sidebar>) => {
    const [isOffline, setIsOffline] = useState(!navigator.onLine);

    useEffect(() => {
      const handleOnline = () => setIsOffline(false);
      const handleOffline = () => setIsOffline(true);

      window.addEventListener("online", handleOnline);
      window.addEventListener("offline", handleOffline);

      return () => {
        window.removeEventListener("online", handleOnline);
        window.removeEventListener("offline", handleOffline);
      };
    }, []);

    return (
      <Sidebar collapsible="icon" {...props}>
        <SidebarHeader></SidebarHeader>
        <SidebarContent>
          <NavMain items={data.navMain} />
        </SidebarContent>
        <SidebarFooter>
          {isOffline && (
            <span
              role="status"
              aria-live="polite"
              className="text-sm text-yellow-600 bg-yellow-100 px-2 py-1 rounded text-center"
            >
              Offline Mode
            </span>
          )}
          <NavUser />
        </SidebarFooter>
        <SidebarRail />
      </Sidebar>
    );
  },
);

export { ConsoleSidebar };
