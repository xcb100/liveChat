<template>
  <div class="app-page auth-page">
    <section class="auth-hero">
      <div>
        <span class="auth-kicker">LiveChat Console</span>
        <h1 class="auth-title">客服登录</h1>
      </div>
    </section>

    <section class="auth-panel">
      <div class="auth-panel-header">
        <div>
          <h2 class="auth-panel-title">{{ activeMode === "login" ? "登录工作台" : "注册客服账号" }}</h2>
          <p class="auth-panel-subtitle">
            {{ activeMode === "login" ? "使用现有账号登录。" : "创建账号后返回登录页。" }}
          </p>
        </div>
        <button class="auth-switch" type="button" @click="switchMode(activeMode === 'login' ? 'register' : 'login')">
          {{ activeMode === "login" ? "创建账号" : "返回登录" }}
        </button>
      </div>

      <InlineAlert :message="feedback.message" :variant="feedback.type" />

      <form class="auth-form" @submit.prevent="handleSubmit">
        <div class="auth-field">
          <label for="account">账号</label>
          <input id="account" v-model.trim="form.account" autocomplete="username" placeholder="输入账号">
          <span v-if="fieldErrors.account" class="auth-error-text">{{ fieldErrors.account }}</span>
        </div>

        <div v-if="activeMode === 'register'" class="auth-field">
          <label for="nickname">昵称</label>
          <input id="nickname" v-model.trim="form.nickname" autocomplete="nickname" placeholder="输入客服显示昵称">
          <span v-if="fieldErrors.nickname" class="auth-error-text">{{ fieldErrors.nickname }}</span>
        </div>

        <div class="auth-field">
          <label for="password">密码</label>
          <input id="password" v-model="form.password" :autocomplete="activeMode === 'login' ? 'current-password' : 'new-password'" type="password" placeholder="输入密码">
          <span v-if="fieldErrors.password" class="auth-error-text">{{ fieldErrors.password }}</span>
        </div>

        <div v-if="activeMode === 'register'" class="auth-field">
          <label for="rePassword">确认密码</label>
          <input id="rePassword" v-model="form.rePassword" autocomplete="new-password" type="password" placeholder="再次输入密码">
          <span v-if="fieldErrors.rePassword" class="auth-error-text">{{ fieldErrors.rePassword }}</span>
        </div>

        <div class="auth-button-row">
          <button class="primary-button" :disabled="isSubmitting" type="submit">
            {{ isSubmitting ? "处理中..." : activeMode === "login" ? "进入工作台" : "创建账号" }}
          </button>
          <button v-if="activeMode === 'register'" class="secondary-button" :disabled="isSubmitting" type="button" @click="switchMode('login')">
            返回登录
          </button>
        </div>
      </form>

    </section>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

import InlineAlert from "@/components/InlineAlert.vue";
import { createFormData, requestJsonWithTimeout } from "@/services/http";
import { breakOutOfIframe, setAuthToken } from "@/services/session";

const route = useRoute();
const router = useRouter();

const activeMode = ref("login");
const isSubmitting = ref(false);
const feedback = reactive({
  type: "",
  message: "",
});
const form = reactive({
  account: "",
  nickname: "",
  password: "",
  rePassword: "",
});
const fieldErrors = reactive({
  account: "",
  nickname: "",
  password: "",
  rePassword: "",
});

// resetFeedback 输入为空，输出为提示状态清理结果，目的在于在新提交前移除旧提示和字段错误。
function resetFeedback() {
  feedback.type = "";
  feedback.message = "";
  fieldErrors.account = "";
  fieldErrors.nickname = "";
  fieldErrors.password = "";
  fieldErrors.rePassword = "";
}

// switchMode 输入目标模式，输出为界面模式切换结果，目的在于在登录和注册之间复用同一页结构。
function switchMode(nextMode) {
  resetFeedback();
  activeMode.value = nextMode;
}

// validateLoginForm 输入为空，输出为登录表单是否合法，目的在于在请求前完成必要字段校验。
function validateLoginForm() {
  resetFeedback();
  let isValid = true;

  if (!form.account.trim()) {
    fieldErrors.account = "请输入账号";
    isValid = false;
  }

  if (!form.password) {
    fieldErrors.password = "请输入密码";
    isValid = false;
  }

  return isValid;
}

// validateRegisterForm 输入为空，输出为注册表单是否合法，目的在于在请求前完成账号与密码约束检查。
function validateRegisterForm() {
  resetFeedback();
  let isValid = true;

  if (form.account.trim().length < 2 || form.account.trim().length > 20) {
    fieldErrors.account = "账号长度需要在 2 到 20 个字符之间";
    isValid = false;
  }

  if (form.nickname.trim().length < 1 || form.nickname.trim().length > 60) {
    fieldErrors.nickname = "昵称长度需要在 1 到 60 个字符之间";
    isValid = false;
  }

  if (form.password.length < 2) {
    fieldErrors.password = "密码至少需要 2 个字符";
    isValid = false;
  }

  if (form.password !== form.rePassword) {
    fieldErrors.rePassword = "两次输入的密码不一致";
    isValid = false;
  }

  return isValid;
}

// submitLogin 输入为空，输出为登录结果，目的在于保留原有鉴权接口并在成功后跳转工作台。
async function submitLogin() {
  if (!validateLoginForm()) {
    return;
  }

  isSubmitting.value = true;
  try {
    const payload = await requestJsonWithTimeout("/check", {
      method: "POST",
      body: createFormData({
        username: form.account.trim(),
        password: form.password,
      }),
    });

    if (Number(payload.code) !== 200) {
      feedback.type = "error";
      feedback.message = payload.message || payload.msg || "登录失败";
      return;
    }

    setAuthToken(payload.result.token);
    const redirectPath = typeof route.query.redirect === "string" ? route.query.redirect : "/main";
    await router.replace(redirectPath);
  } catch (error) {
    feedback.type = "error";
    feedback.message = error.message || "登录请求失败，请稍后重试";
  } finally {
    isSubmitting.value = false;
  }
}

// submitRegister 输入为空，输出为注册结果，目的在于保留原有注册接口并在成功后切回登录流程。
async function submitRegister() {
  if (!validateRegisterForm()) {
    return;
  }

  isSubmitting.value = true;
  try {
    const payload = await requestJsonWithTimeout("/register", {
      method: "POST",
      body: createFormData({
        username: form.account.trim(),
        nickname: form.nickname.trim(),
        password: form.password,
      }),
    });

    if (Number(payload.code) !== 200) {
      feedback.type = "error";
      feedback.message = payload.message || payload.msg || "注册失败";
      return;
    }

    feedback.type = "success";
    feedback.message = "账号创建完成，现在可以直接登录工作台";
    activeMode.value = "login";
    form.password = "";
    form.rePassword = "";
  } catch (error) {
    feedback.type = "error";
    feedback.message = error.message || "注册请求失败，请稍后重试";
  } finally {
    isSubmitting.value = false;
  }
}

// handleSubmit 输入为空，输出为当前模式对应的提交结果，目的在于统一按钮触发逻辑。
function handleSubmit() {
  if (activeMode.value === "login") {
    submitLogin();
    return;
  }
  submitRegister();
}

onMounted(() => {
  breakOutOfIframe();
});
</script>
