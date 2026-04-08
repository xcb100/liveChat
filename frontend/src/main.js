import { createApp } from "vue";

import App from "./App.vue";
import router from "./router";
import "./styles/base.css";

// bootstrapApplication 输入为空，输出为 Vue 应用挂载结果，目的在于统一初始化路由和全局样式。
function bootstrapApplication() {
  const application = createApp(App);
  application.use(router);
  application.mount("#app");
}

bootstrapApplication();
