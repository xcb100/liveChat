<template>
  <div class="app-page auth-page">
    <section class="auth-hero">
      <div>
        <span class="auth-kicker">Installation</span>
        <h1 class="auth-title">初始化数据库</h1>
      </div>
    </section>

    <section class="auth-panel">
      <div class="auth-panel-header">
        <div>
          <h2 class="auth-panel-title">初始化数据库</h2>
          <p class="auth-panel-subtitle">填写数据库连接信息后执行安装。</p>
        </div>
      </div>

      <InlineAlert :message="feedback.message" :variant="feedback.type" />

      <form class="auth-form" @submit.prevent="submitInstall">
        <div class="auth-field">
          <label for="server">数据库地址</label>
          <input id="server" v-model.trim="mysql.server" placeholder="例如：127.0.0.1">
          <span v-if="fieldErrors.server" class="auth-error-text">{{ fieldErrors.server }}</span>
        </div>

        <div class="auth-field">
          <label for="port">数据库端口</label>
          <input id="port" v-model.trim="mysql.port" placeholder="例如：3306">
          <span v-if="fieldErrors.port" class="auth-error-text">{{ fieldErrors.port }}</span>
        </div>

        <div class="auth-field">
          <label for="database">数据库名</label>
          <input id="database" v-model.trim="mysql.database" placeholder="输入数据库名">
          <span v-if="fieldErrors.database" class="auth-error-text">{{ fieldErrors.database }}</span>
        </div>

        <div class="auth-field">
          <label for="username">数据库用户名</label>
          <input id="username" v-model.trim="mysql.username" placeholder="输入数据库用户名">
          <span v-if="fieldErrors.username" class="auth-error-text">{{ fieldErrors.username }}</span>
        </div>

        <div class="auth-field">
          <label for="password">数据库密码</label>
          <input id="password" v-model="mysql.password" type="password" placeholder="输入数据库密码">
          <span v-if="fieldErrors.password" class="auth-error-text">{{ fieldErrors.password }}</span>
        </div>

        <div class="auth-button-row">
          <button class="primary-button" :disabled="isSubmitting" type="submit">
            {{ isSubmitting ? "安装中..." : "执行安装" }}
          </button>
          <RouterLink class="auth-link" to="/login">已经安装过，直接去登录</RouterLink>
        </div>
      </form>
    </section>
  </div>
</template>

<script setup>
import { reactive, ref } from "vue";

import InlineAlert from "@/components/InlineAlert.vue";
import { createFormData, requestJsonWithTimeout } from "@/services/http";

const isSubmitting = ref(false);
const feedback = reactive({
  type: "",
  message: "",
});
const fieldErrors = reactive({
  server: "",
  port: "",
  database: "",
  username: "",
  password: "",
});
const mysql = reactive({
  server: "",
  port: "3306",
  database: "",
  username: "",
  password: "",
});

// resetFeedback 输入为空，输出为安装提示状态清理结果，目的在于每次提交前移除旧提示和字段错误。
function resetFeedback() {
  feedback.type = "";
  feedback.message = "";
  fieldErrors.server = "";
  fieldErrors.port = "";
  fieldErrors.database = "";
  fieldErrors.username = "";
  fieldErrors.password = "";
}

// validateInstallForm 输入为空，输出为安装表单是否合法，目的在于确保数据库配置完整后再发起安装。
function validateInstallForm() {
  resetFeedback();
  let isValid = true;

  Object.entries(mysql).forEach(([key, value]) => {
    if (!String(value || "").trim()) {
      fieldErrors[key] = "该字段不能为空";
      isValid = false;
    }
  });

  return isValid;
}

// submitInstall 输入为空，输出为安装结果，目的在于沿用原有安装接口完成首轮部署。
async function submitInstall() {
  if (!validateInstallForm()) {
    return;
  }

  isSubmitting.value = true;
  try {
    const payload = await requestJsonWithTimeout("/install", {
      method: "POST",
      body: createFormData(mysql),
    });
    feedback.type = Number(payload.code) === 200 ? "success" : "error";
    feedback.message = payload.msg || payload.message || "安装请求已完成";
  } catch (error) {
    feedback.type = "error";
    feedback.message = error.message || "安装请求失败，请检查网络与服务状态";
  } finally {
    isSubmitting.value = false;
  }
}
</script>
