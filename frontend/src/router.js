import { createRouter, createWebHistory } from "vue-router";

import { clearAuthToken, getAuthToken } from "@/services/session";
import InstallView from "@/views/InstallView.vue";
import LiveChatView from "@/views/LiveChatView.vue";
import LoginView from "@/views/LoginView.vue";
import WorkspaceView from "@/views/WorkspaceView.vue";

const routes = [
  { path: "/", redirect: "/login" },
  {
    path: "/login",
    name: "login",
    component: LoginView,
    meta: {
      title: "LiveChat 登录",
      guestOnly: true,
    },
  },
  {
    path: "/install",
    name: "install",
    component: InstallView,
    meta: {
      title: "LiveChat 安装",
    },
  },
  {
    path: "/main",
    name: "workspace",
    component: WorkspaceView,
    meta: {
      title: "LiveChat 工作台",
      requiresAuth: true,
    },
  },
  {
    path: "/livechat",
    name: "livechat",
    component: LiveChatView,
    meta: {
      title: "LiveChat 会话",
    },
  },
  { path: "/pannel", redirect: "/main" },
  { path: "/chat_main", redirect: "/main" },
  { path: "/setting", redirect: "/main?panel=profile" },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

// updateDocumentTitle 输入目标路由，输出为标题更新结果，目的在于让不同页面保持清晰的浏览器标题。
function updateDocumentTitle(toRoute) {
  document.title = toRoute.meta?.title || "LiveChat";
}

router.beforeEach((to, from, next) => {
  updateDocumentTitle(to);
  const authToken = getAuthToken();

  if (to.meta?.requiresAuth && !authToken) {
    next({ path: "/login", query: { redirect: to.fullPath } });
    return;
  }

  if (to.meta?.guestOnly && authToken) {
    next("/main");
    return;
  }

  next();
});

router.afterEach((to) => {
  if (to.meta?.guestOnly && !getAuthToken()) {
    clearAuthToken();
  }
});

export default router;
