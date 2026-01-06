import Layout from "@/business/app-layout/horizontal-layout"

const Dashboard = {
    path: "/dashboard",
    sort: 0,
    global: true,
    component: Layout,
    name: "Dashboards",
    children: [
        {
            path: "/dashboard",
            component: () => import("@/business/dashboard"),
            name: "Dashboard",
            meta: {
                title: "business.dashboard.dashboard",
                icon: "iconfont icongailan"
            },
        },
        {
            path: "/dashboard/route-matcher",
            component: () => import("@/business/dashboard/route-matcher"),
            name: "RouteMatcher",
            hidden: true,
            meta: {
                title: "business.tools.route_matcher.title",
                activeMenu: "/dashboard",
            },
        },
    ]
}

export default Dashboard
