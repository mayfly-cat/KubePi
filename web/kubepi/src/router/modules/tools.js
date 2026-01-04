import Layout from "@/business/app-layout/horizontal-layout";

const Tools = {
  path: "/tools",
  sort: 5,
  component: Layout,
  name: "Tools",
  meta: {
    title: "business.tools.title",
    icon: "el-icon-s-tools",
  },
  children: [
    {
      path: "/tools",
      component: () => import("@/business/tools/index"),
      name: "ToolsIndex",
      meta: {
        title: "business.tools.title",
        activeMenu: "/tools",
      },
    },
  ],
};

export default Tools;

