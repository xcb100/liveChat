<template>
  <div v-if="isLoading" class="app-page workspace-loading">工作台加载中…</div>
  <div v-else class="app-page">
    <div class="workspace-shell">
      <aside class="workspace-panel workspace-column">
        <div class="workspace-header">
          <div class="workspace-brand">
            <img class="workspace-avatar is-large" :src="profile.avator || '/static/images/admin.png'" alt="avatar">
            <div>
              <h1 class="workspace-title">LiveChat 工作台</h1>
              <p class="workspace-subtitle">{{ profile.nickname || profile.username }}</p>
            </div>
          </div>
        </div>
        <div class="workspace-stat-grid">
          <div v-for="item in statCards" :key="item.label" class="workspace-stat-card">
            <span class="workspace-stat-label">{{ item.label }}</span>
            <span class="workspace-stat-value">{{ item.value }}</span>
          </div>
        </div>
        <div class="workspace-tabs">
          <button class="tab-button" :class="{ 'is-active': leftPanel === 'online' }" @click="leftPanel = 'online'">在线访客</button>
          <button class="tab-button" :class="{ 'is-active': leftPanel === 'recent' }" @click="leftPanel = 'recent'">最近访客</button>
        </div>
        <div class="workspace-scroll workspace-section">
          <button class="secondary-button" type="button" @click="bootstrapWorkbench">刷新概览</button>
          <button
            v-for="visitor in visibleVisitors"
            :key="visitor.uid || visitor.visitor_id"
            class="workspace-list-card is-clickable"
            :class="{ 'is-selected': currentVisitorId === (visitor.uid || visitor.visitor_id) }"
            type="button"
            @click="selectVisitor(visitor.uid || visitor.visitor_id)"
          >
            <div class="workspace-list-row">
              <img class="workspace-avatar" :src="visitor.avator || '/static/images/2.png'" alt="visitor">
              <div class="workspace-list-meta">
                <div class="workspace-list-title">
                  <span>{{ visitor.username || visitor.name }}</span>
                  <span class="pill" :class="Number(visitor.status || 1) === 1 ? 'is-success' : 'is-accent'">{{ leftPanel === 'online' ? '在线' : Number(visitor.status || 0) === 1 ? '在线' : '历史' }}</span>
                </div>
                <div class="workspace-last-message">{{ visitor.last_message || '暂无消息摘要' }}</div>
              </div>
            </div>
          </button>
          <div v-if="visibleVisitors.length === 0" class="workspace-empty">当前没有可展示的会话。</div>
          <div v-if="leftPanel === 'recent'" class="workspace-inline-actions">
            <button class="secondary-button" :disabled="pagination.page <= 1" type="button" @click="loadRecentVisitors(pagination.page - 1)">上一页</button>
            <span class="workspace-muted">第 {{ pagination.page }} 页 / 共 {{ pageCount }} 页</span>
            <button class="secondary-button" :disabled="pagination.page >= pageCount" type="button" @click="loadRecentVisitors(pagination.page + 1)">下一页</button>
          </div>
        </div>
      </aside>

      <main class="workspace-panel workspace-chat-panel">
        <div class="workspace-chat-header">
          <div>
            <h2 class="workspace-title">{{ currentVisitorTitle }}</h2>
            <p class="workspace-subtitle">{{ currentVisitorSubtitle }}</p>
          </div>
          <div class="workspace-inline-actions">
            <span class="status-chip" :class="statusChipClass"><span class="status-dot"></span>{{ connectionLabel }}</span>
            <button class="secondary-button" :disabled="!currentVisitorId" type="button" @click="transferConversation">转接</button>
            <button class="danger-button" :disabled="!currentVisitorId" type="button" @click="closeConversation">结束会话</button>
          </div>
        </div>
        <InlineAlert :message="toastMessage" variant="info" />
        <div v-if="currentVisitorId" class="workspace-chat-area">
          <div ref="messageAreaRef" class="workspace-chat-scroll message-list">
            <button v-if="hasMoreHistory" class="secondary-button load-more-button" type="button" @click="loadMoreMessages">加载更早消息</button>
            <div v-for="message in messages" :key="message.key" class="message-item" :class="{ 'is-self': message.isSelf }">
              <img class="chat-avatar" :src="message.avator" alt="avatar">
              <div class="message-body">
                <div class="message-name">{{ message.name }}</div>
                <div class="message-bubble" v-html="message.renderedContent"></div>
                <div class="message-time">{{ message.time }}</div>
              </div>
            </div>
          </div>
          <div class="composer-card">
            <div class="composer-toolbar">
              <div class="workspace-inline-actions">
                <button class="icon-button" type="button" @click="triggerFileInput('image')">图</button>
                <button class="icon-button" type="button" @click="triggerFileInput('attachment')">文</button>
              </div>
              <div class="workspace-muted">{{ typingNotice || 'Enter 发送，Shift + Enter 换行' }}</div>
            </div>
            <textarea class="panel-textarea" v-model="composer" placeholder="输入消息内容，支持图片、附件协议和快捷回复。" @keydown="handleComposerKeydown"></textarea>
            <div class="panel-actions" style="margin-top: 12px;">
              <button class="secondary-button" type="button" @click="composer = ''">清空</button>
              <button class="primary-button" :disabled="isSending || !composer.trim()" type="button" @click="sendMessage">发送消息</button>
            </div>
          </div>
        </div>
        <div v-else class="workspace-empty">
          <p class="workspace-muted">左侧选择一个会话开始处理消息。</p>
        </div>
      </main>

      <aside class="workspace-panel workspace-column">
        <div class="workspace-side-header">
          <div>
            <h3 class="workspace-title" style="font-size: 22px;">辅助面板</h3>
            <p class="workspace-subtitle">会话相关信息</p>
          </div>
          <button class="ghost-button" type="button" @click="logout">退出登录</button>
        </div>
        <div class="workspace-side-tabs">
          <button class="tab-button" :class="{ 'is-active': rightPanel === 'replies' }" @click="rightPanel = 'replies'">快捷回复</button>
          <button class="tab-button" :class="{ 'is-active': rightPanel === 'visitor' }" @click="rightPanel = 'visitor'">访客信息</button>
          <button class="tab-button" :class="{ 'is-active': rightPanel === 'agent' }" @click="rightPanel = 'agent'">Agent</button>
          <button class="tab-button" :class="{ 'is-active': rightPanel === 'profile' }" @click="rightPanel = 'profile'">个人资料</button>
        </div>
        <div class="workspace-side-scroll workspace-section">
          <template v-if="rightPanel === 'replies'">
            <input v-model.trim="replyQuery" class="workspace-search" placeholder="筛选快捷回复">
            <div class="workspace-section">
              <div v-for="group in filteredReplyGroups" :key="group.group_id" class="workspace-card">
                <strong>{{ group.group_name }}</strong>
                <div class="workspace-section" style="margin-top: 10px;">
                  <div v-for="item in group.items" :key="item.item_id" class="reply-card">
                    <div class="reply-title">{{ item.item_name }}</div>
                    <div class="reply-preview">{{ item.item_content }}</div>
                    <div class="workspace-inline-actions" style="margin-top: 10px;">
                      <button class="ghost-button" type="button" @click="applyReply(item.item_content)">填入输入框</button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </template>
          <template v-else-if="rightPanel === 'visitor'">
            <div v-if="visitorDetail.visitor_id" class="workspace-detail-grid">
              <div v-for="item in visitorDetails" :key="item.label" class="workspace-detail-item">
                <div class="workspace-detail-label">{{ item.label }}</div>
                <div class="workspace-detail-value">{{ item.value }}</div>
              </div>
              <div v-for="item in visitorExtraFields" :key="item.key" class="workspace-detail-item">
                <div class="workspace-detail-label">{{ item.key }}</div>
                <div class="workspace-detail-value">{{ item.value }}</div>
              </div>
            </div>
            <div v-else class="workspace-empty">选择会话后展示访客详情。</div>
            <div class="panel-actions">
              <button class="secondary-button" :disabled="!visitorDetail.source_ip" type="button" @click="addBlacklist(visitorDetail.source_ip)">加入黑名单</button>
            </div>
            <div class="workspace-section">
              <div v-for="item in blacklist" :key="item.ip" class="workspace-list-card">
                <div class="workspace-list-row">
                  <div class="workspace-list-meta">
                    <div class="workspace-list-title"><span>{{ item.ip }}</span><span class="pill is-danger">已限制</span></div>
                  </div>
                  <button class="ghost-button" type="button" @click="removeBlacklist(item.ip)">移除</button>
                </div>
              </div>
            </div>
          </template>
          <template v-else-if="rightPanel === 'agent'">
            <div class="workspace-card" v-for="item in agentOverview" :key="item.agent_id">
              <div class="reply-header">
                <div>
                  <div class="reply-title">{{ item.display_name || item.agent_id }}</div>
                  <div class="reply-preview">{{ item.agent_id }}</div>
                </div>
                <span class="pill" :class="item.available ? 'is-success' : 'is-accent'">{{ item.available ? '可接待' : '繁忙/离线' }}</span>
              </div>
              <div class="workspace-inline-actions" style="margin-top: 10px;">
                <span class="pill">进行中 {{ item.active_sessions }}</span>
                <span class="pill">剩余容量 {{ item.available_sessions }}</span>
                <span v-for="capability in formatAgentCapabilities(item.capabilities)" :key="`${item.agent_id}-${capability}`" class="pill">{{ capability }}</span>
              </div>
            </div>
            <div v-if="agentOverview.length === 0" class="workspace-empty">当前没有已注册 agent。</div>
          </template>
          <template v-else>
            <div class="auth-field"><label>账号</label><input class="panel-input" :value="profile.username" disabled></div>
            <div class="auth-field"><label>昵称</label><input v-model.trim="profileForm.nickname" class="panel-input" placeholder="客服显示昵称"></div>
            <div class="auth-field"><label>头像地址</label><input v-model.trim="profileForm.avator" class="panel-input" placeholder="/static/images/admin.png"></div>
            <div class="panel-actions">
              <button class="secondary-button" type="button" @click="triggerFileInput('avatar')">上传头像</button>
              <button class="primary-button" type="button" @click="saveProfile">保存资料</button>
            </div>
            <div class="auth-field"><label>旧密码</label><input v-model="profileForm.oldPassword" class="panel-input" type="password"></div>
            <div class="auth-field"><label>新密码</label><input v-model="profileForm.newPassword" class="panel-input" type="password"></div>
            <div class="auth-field"><label>确认新密码</label><input v-model="profileForm.confirmPassword" class="panel-input" type="password"></div>
            <button class="primary-button" type="button" @click="updatePassword">修改密码</button>
          </template>
        </div>
      </aside>
    </div>

    <input ref="imageInputRef" type="file" accept="image/gif,image/jpeg,image/jpg,image/png" hidden @change="handleImageUpload">
    <input ref="attachmentInputRef" type="file" hidden @change="handleAttachmentUpload">
    <input ref="avatarInputRef" type="file" accept="image/gif,image/jpeg,image/jpg,image/png" hidden @change="handleAvatarUpload">
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import InlineAlert from "@/components/InlineAlert.vue";
import { createAuthorizedOptions, createFormData, requestJson } from "@/services/http";
import { clearAuthToken, getAuthToken } from "@/services/session";
import { base64ToUtf8, formatDate, getWsBaseUrl, notify, renderChatContent } from "@/utils/chat";

const route = useRoute();
const router = useRouter();
const isLoading = ref(true);
const isSending = ref(false);
const leftPanel = ref("online");
const rightPanel = ref(route.query.panel === "profile" ? "profile" : "replies");
const connectionStatus = ref("connecting");
const currentVisitorId = ref("");
const composer = ref("");
const typingNotice = ref("");
const toastMessage = ref("");
const hasMoreHistory = ref(false);
const replyQuery = ref("");
const historyPage = ref(1);
const messageAreaRef = ref(null);
const imageInputRef = ref(null);
const attachmentInputRef = ref(null);
const avatarInputRef = ref(null);
const onlineVisitors = ref([]);
const recentVisitors = ref([]);
const replyGroups = ref([]);
const blacklist = ref([]);
const agentOverview = ref([]);
const messages = ref([]);
const pagination = reactive({ count: 0, page: 1, pagesize: 8 });
const metrics = reactive({ online_count: 0, recent_count: 0, reply_count: 0, blacklist_count: 0, agent_total_count: 0, agent_available_count: 0 });
const profile = reactive({ uid: 0, username: "", nickname: "", avator: "" });
const profileForm = reactive({ nickname: "", avator: "", oldPassword: "", newPassword: "", confirmPassword: "" });
const visitorDetail = reactive({ visitor_id: "", name: "", source_ip: "", client_ip: "", city: "", refer: "", created_at: "", updated_at: "", status: "", extra: "" });
let socketInstance = null;
let heartbeatTimer = 0;
let reconnectTimer = 0;
let lastTypingAt = 0;

const pageCount = computed(() => Math.max(1, Math.ceil((pagination.count || 1) / Math.max(1, pagination.pagesize))));
const visibleVisitors = computed(() => leftPanel.value === "online" ? onlineVisitors.value : recentVisitors.value);
const statCards = computed(() => [
  { label: "在线会话", value: metrics.online_count },
  { label: "快捷回复组", value: metrics.reply_count },
  { label: "最近访客", value: metrics.recent_count },
  { label: "黑名单", value: metrics.blacklist_count },
  { label: "Agent 总数", value: metrics.agent_total_count },
  { label: "可接待 Agent", value: metrics.agent_available_count },
]);
const filteredReplyGroups = computed(() => {
  if (!replyQuery.value.trim()) return replyGroups.value;
  const keyword = replyQuery.value.trim().toLowerCase();
  return replyGroups.value.map((group) => ({ ...group, items: (group.items || []).filter((item) => `${item.item_name} ${item.item_content}`.toLowerCase().includes(keyword)) })).filter((group) => group.items.length > 0);
});
const visitorExtraFields = computed(() => decodeVisitorExtra(visitorDetail.extra));
const visitorDetails = computed(() => [
  { label: "来源 IP", value: visitorDetail.source_ip || "未知" },
  { label: "地区", value: visitorDetail.city || "未知" },
  { label: "首次访问", value: visitorDetail.created_at || "未知" },
  { label: "最后活跃", value: visitorDetail.updated_at || "未知" },
  { label: "来源页面", value: visitorDetail.refer || "直接访问" },
]);
const currentVisitorTitle = computed(() => visitorDetail.name || "请选择一个会话");
const currentVisitorSubtitle = computed(() => visitorDetail.visitor_id ? `${visitorDetail.status} · ${visitorDetail.city || "未知地区"} · ${visitorDetail.client_ip || "未知 IP"}` : "请选择左侧会话");
const connectionLabel = computed(() => connectionStatus.value === "online" ? "实时连接正常" : connectionStatus.value === "connecting" ? "实时连接建立中" : "实时连接断开，正在重试");
const statusChipClass = computed(() => connectionStatus.value === "online" ? "" : connectionStatus.value === "connecting" ? "is-warning" : "is-danger");

// showToast 输入提示文本，输出为提示状态，目的在于统一反馈当前操作结果。
function showToast(messageText) { toastMessage.value = messageText; window.setTimeout(() => { if (toastMessage.value === messageText) toastMessage.value = ""; }, 2400); }
// decodeVisitorExtra 输入扩展字段，输出为键值数组，目的在于把访客扩展信息转换为面板可读内容。
function decodeVisitorExtra(extraValue) { if (!extraValue) return []; try { const parsedValue = JSON.parse(base64ToUtf8(extraValue)); return Object.keys(parsedValue).filter((key) => !["visitorAvatar", "visitorName"].includes(key)).map((key) => ({ key, value: parsedValue[key] || "无" })); } catch { return []; } }
// normalizeMessageRecord 输入原始消息，输出为标准消息对象，目的在于统一历史消息和实时消息的渲染格式。
function normalizeMessageRecord(messageRecord) { const rawFlag = String(messageRecord.is_kefu ?? messageRecord.mes_type ?? "").toLowerCase(); const isSelf = rawFlag === "yes" || rawFlag === "true" || rawFlag === "kefu"; return { key: `${messageRecord.id || messageRecord.visitor_id || messageRecord.to_id || Date.now()}-${Math.random()}`, name: isSelf ? messageRecord.name || messageRecord.kefu_name || profile.nickname || profile.username : messageRecord.name || messageRecord.visitor_name || "访客", avator: isSelf ? messageRecord.avator || messageRecord.kefu_avator || profile.avator || "/static/images/admin.png" : messageRecord.avator || messageRecord.visitor_avator || "/static/images/2.png", time: messageRecord.time || messageRecord.create_time || formatDate(new Date().toISOString()), content: messageRecord.content || "", isSelf, renderedContent: renderChatContent(messageRecord.content || "") }; }
// sortVisitors 输入访客列表，输出为排序结果，目的在于让摘要按最后活跃时间稳定排序。
function sortVisitors(visitorList) { return [...visitorList].sort((left, right) => (right.updated_at || 0) - (left.updated_at || 0)); }
// updatePreview 输入列表、访客标识和消息内容，输出为空，目的在于同步侧边栏摘要。
function updatePreview(collection, visitorId, content) { const matchedItem = collection.find((item) => item.uid === visitorId || item.visitor_id === visitorId); if (matchedItem) matchedItem.last_message = content; }
// scrollToBottom 输入为空，输出为滚动结果，目的在于让聊天区保持聚焦最新消息。
async function scrollToBottom() { await nextTick(); if (messageAreaRef.value) messageAreaRef.value.scrollTop = messageAreaRef.value.scrollHeight; }
// bootstrapWorkbench 输入为空，输出为首屏初始化结果，目的在于拉取工作台所需主要数据。
async function bootstrapWorkbench() { const result = await requestJson("/workbench/bootstrap", createAuthorizedOptions()); Object.assign(profile, result.profile || {}); profileForm.nickname = result.profile?.nickname || ""; profileForm.avator = result.profile?.avator || ""; onlineVisitors.value = sortVisitors(result.online_visitors || []); recentVisitors.value = result.recent_visitors?.list || []; replyGroups.value = result.reply_groups || []; blacklist.value = result.blacklists || []; agentOverview.value = result.agent_overview || []; Object.assign(metrics, result.metrics || {}); pagination.count = result.recent_visitors?.count || 0; pagination.page = result.recent_visitors?.page || 1; pagination.pagesize = result.recent_visitors?.pagesize || 8; if (!currentVisitorId.value && onlineVisitors.value.length > 0 && rightPanel.value !== "profile") await selectVisitor(onlineVisitors.value[0].uid); }
// loadRecentVisitors 输入页码，输出为历史访客加载结果，目的在于支持分页浏览最近会话。
async function loadRecentVisitors(page) { const result = await requestJson(`/visitors?page=${page}&pagesize=${pagination.pagesize}`, createAuthorizedOptions()); recentVisitors.value = result.list || []; pagination.count = result.count || 0; pagination.page = page; pagination.pagesize = result.pagesize || pagination.pagesize; }
// loadVisitorDetail 输入访客标识，输出为访客详情结果，目的在于刷新右侧访客详情面板。
async function loadVisitorDetail(visitorId) { const result = await requestJson(`/visitor?visitorId=${encodeURIComponent(visitorId)}`, createAuthorizedOptions()); Object.assign(visitorDetail, { visitor_id: result.visitor_id || "", name: result.name || "", source_ip: result.source_ip || "", client_ip: result.client_ip || "", city: result.city || "", refer: result.refer || "", created_at: result.created_at ? formatDate(result.created_at) : "", updated_at: result.updated_at ? formatDate(result.updated_at) : "", status: Number(result.status) === 1 ? "在线" : "离线", extra: result.extra || "" }); }
// loadMessages 输入访客标识和重置标记，输出为消息列表结果，目的在于按页读取当前会话消息。
async function loadMessages(visitorId, reset = true) { if (reset) { historyPage.value = 1; messages.value = []; } const result = await requestJson(`/2/messagesPages?visitor_id=${encodeURIComponent(visitorId)}&page=${historyPage.value}&pagesize=20`, createAuthorizedOptions()); const normalizedMessages = (result.list || []).slice().reverse().map((item) => normalizeMessageRecord(item)); messages.value = reset ? normalizedMessages : [...normalizedMessages, ...messages.value]; hasMoreHistory.value = messages.value.length < (result.count || 0); if (reset) await scrollToBottom(); }
// selectVisitor 输入访客标识，输出为会话切换结果，目的在于更新当前聊天上下文。
async function selectVisitor(visitorId) { currentVisitorId.value = visitorId; typingNotice.value = ""; await Promise.all([loadVisitorDetail(visitorId), loadMessages(visitorId, true)]); }
// loadMoreMessages 输入为空，输出为更多历史消息结果，目的在于向前翻页加载会话内容。
async function loadMoreMessages() { if (!currentVisitorId.value || !hasMoreHistory.value) return; historyPage.value += 1; await loadMessages(currentVisitorId.value, false); }
// startHeartbeat 输入为空，输出为心跳启动结果，目的在于维持客服端 WebSocket 活跃。
function startHeartbeat() { stopHeartbeat(); heartbeatTimer = window.setInterval(() => { if (socketInstance?.readyState === WebSocket.OPEN) socketInstance.send(JSON.stringify({ type: "ping", data: "" })); }, 25000); }
// stopHeartbeat 输入为空，输出为心跳停止结果，目的在于避免重复定时器泄漏。
function stopHeartbeat() { if (heartbeatTimer) { window.clearInterval(heartbeatTimer); heartbeatTimer = 0; } }
// scheduleReconnect 输入为空，输出为重连调度结果，目的在于连接异常后自动恢复实时能力。
function scheduleReconnect() { if (reconnectTimer) return; reconnectTimer = window.setTimeout(() => { reconnectTimer = 0; openSocket(); }, 3000); }
// handleSocketMessage 输入消息文本，输出为状态更新结果，目的在于统一处理客服端实时事件。
async function handleSocketMessage(rawMessage) { const payload = JSON.parse(rawMessage); if (["pong", "many pong"].includes(payload.type)) return; if (payload.type === "inputing") { if (payload.data?.from === currentVisitorId.value) { typingNotice.value = "对方正在输入…"; window.setTimeout(() => { if (typingNotice.value === "对方正在输入…") typingNotice.value = ""; }, 1200); } return; } if (payload.type === "userOnline") { const item = { uid: payload.data.uid, username: payload.data.username, avator: payload.data.avator, last_message: payload.data.last_message || "new visitor", updated_at: Date.now() }; const exists = onlineVisitors.value.find((visitor) => visitor.uid === item.uid); if (exists) Object.assign(exists, item); else onlineVisitors.value = [item, ...onlineVisitors.value]; onlineVisitors.value = sortVisitors(onlineVisitors.value); metrics.online_count = onlineVisitors.value.length; return; } if (payload.type === "userOffline") { onlineVisitors.value = onlineVisitors.value.filter((visitor) => visitor.uid !== payload.data?.uid); metrics.online_count = onlineVisitors.value.length; return; } if (payload.type !== "message") return; const normalizedMessage = normalizeMessageRecord(payload.data); const visitorId = payload.data.id || payload.data.visitor_id; updatePreview(onlineVisitors.value, visitorId, normalizedMessage.content); updatePreview(recentVisitors.value, visitorId, normalizedMessage.content); if (visitorId === currentVisitorId.value) { messages.value = [...messages.value, normalizedMessage]; await scrollToBottom(); return; } if (!normalizedMessage.isSelf) notify(payload.data.name || "新消息", { body: normalizedMessage.content, icon: payload.data.avator || "/static/images/2.png" }, () => selectVisitor(visitorId)); }
// openSocket 输入为空，输出为 WebSocket 建连结果，目的在于建立客服工作台实时消息通道。
function openSocket() { const token = getAuthToken(); if (!token) { router.replace("/login"); return; } connectionStatus.value = "connecting"; socketInstance = new WebSocket(`${getWsBaseUrl()}/ws_kefu?token=${encodeURIComponent(token)}`); socketInstance.onopen = () => { connectionStatus.value = "online"; startHeartbeat(); }; socketInstance.onmessage = (event) => { handleSocketMessage(event.data); }; socketInstance.onerror = () => { connectionStatus.value = "offline"; }; socketInstance.onclose = () => { connectionStatus.value = "offline"; stopHeartbeat(); scheduleReconnect(); }; }
// sendMessage 输入为空，输出为消息发送结果，目的在于向当前访客发送客服消息。
async function sendMessage() { if (!currentVisitorId.value || !composer.value.trim()) return; isSending.value = true; try { await requestJson("/kefu/message", createAuthorizedOptions("POST", createFormData({ to_id: currentVisitorId.value, content: composer.value, type: "kefu" }))); composer.value = ""; } catch (error) { showToast(error.message); } finally { isSending.value = false; } }
// closeConversation 输入为空，输出为关闭结果，目的在于主动结束当前访客会话。
async function closeConversation() { if (!currentVisitorId.value) return; try { await requestJson(`/2/message_close?visitor_id=${encodeURIComponent(currentVisitorId.value)}`, createAuthorizedOptions()); showToast("会话已结束"); onlineVisitors.value = onlineVisitors.value.filter((visitor) => visitor.uid !== currentVisitorId.value); currentVisitorId.value = ""; } catch (error) { showToast(error.message); } }
// transferConversation 输入为空，输出为转接结果，目的在于让客服快速把会话转给其他在线客服。
async function transferConversation() { if (!currentVisitorId.value) return; try { const targets = await requestJson("/other_kefulist", createAuthorizedOptions()); const onlineTarget = (targets || []).find((item) => item.status === "online"); if (!onlineTarget) { showToast("当前没有可转接的在线客服"); return; } await requestJson(`/trans_kefu?kefu_id=${encodeURIComponent(onlineTarget.name)}&visitor_id=${encodeURIComponent(currentVisitorId.value)}`, createAuthorizedOptions()); showToast(`已转接给 ${onlineTarget.nickname || onlineTarget.name}`); } catch (error) { showToast(error.message); } }
// saveProfile 输入为空，输出为资料保存结果，目的在于更新当前客服昵称和头像。
async function saveProfile() { try { await requestJson("/kefuinfo", createAuthorizedOptions("POST", createFormData({ nickname: profileForm.nickname, avator: profileForm.avator }))); profile.nickname = profileForm.nickname; profile.avator = profileForm.avator; showToast("资料已保存"); } catch (error) { showToast(error.message); } }
// updatePassword 输入为空，输出为密码更新结果，目的在于通过旧密码校验完成安全改密。
async function updatePassword() { try { await requestJson("/modifypass", createAuthorizedOptions("POST", createFormData({ old_pass: profileForm.oldPassword, new_pass: profileForm.newPassword, confirm_new_pass: profileForm.confirmPassword }))); profileForm.oldPassword = ""; profileForm.newPassword = ""; profileForm.confirmPassword = ""; showToast("密码已更新"); } catch (error) { showToast(error.message); } }
// triggerFileInput 输入类型标识，输出为文件选择动作，目的在于复用隐藏上传控件。
function triggerFileInput(type) { const inputMap = { image: imageInputRef.value, attachment: attachmentInputRef.value, avatar: avatarInputRef.value }; inputMap[type]?.click(); }
// uploadImageAsset 输入图片文件，输出为上传结果，目的在于统一处理工作台图片上传逻辑。
async function uploadImageAsset(file) { const formData = new FormData(); formData.append("imgfile", file); return requestJson("/uploadimg", { method: "POST", body: formData }); }
// handleImageUpload 输入文件事件，输出为图片消息发送结果，目的在于上传图片后直接插入消息协议。
async function handleImageUpload(event) { const file = event.target.files?.[0]; event.target.value = ""; if (!file) return; try { const result = await uploadImageAsset(file); composer.value += `img[/${result.path}]`; await sendMessage(); } catch (error) { showToast(error.message); } }
// handleAttachmentUpload 输入文件事件，输出为附件消息发送结果，目的在于上传附件后插入标准协议文本。
async function handleAttachmentUpload(event) { const file = event.target.files?.[0]; event.target.value = ""; if (!file) return; try { const formData = new FormData(); formData.append("realfile", file); const result = await requestJson("/uploadfile", { method: "POST", body: formData }); composer.value += `attachment[${JSON.stringify({ name: result.name, ext: result.ext, size: result.size, path: `/${result.path}` })}]`; await sendMessage(); } catch (error) { showToast(error.message); } }
// handleAvatarUpload 输入文件事件，输出为头像更新结果，目的在于完成客服头像上传和资料同步。
async function handleAvatarUpload(event) { const file = event.target.files?.[0]; event.target.value = ""; if (!file) return; try { const result = await uploadImageAsset(file); profileForm.avator = `/${result.path}`; await requestJson("/modifyavator", createAuthorizedOptions("POST", createFormData({ avator: profileForm.avator }))); profile.avator = profileForm.avator; showToast("头像已更新"); } catch (error) { showToast(error.message); } }
// addBlacklist 输入 IP 地址，输出为新增结果，目的在于快速限制异常来源访客。
async function addBlacklist(ipAddress) { try { await requestJson("/ipblack", createAuthorizedOptions("POST", createFormData({ ip: ipAddress }))); blacklist.value = await requestJson("/ipblacks", createAuthorizedOptions()); metrics.blacklist_count = blacklist.value.length; showToast("已加入黑名单"); } catch (error) { showToast(error.message); } }
// removeBlacklist 输入 IP 地址，输出为移除结果，目的在于恢复指定来源访问权限。
async function removeBlacklist(ipAddress) { try { await requestJson(`/ipblack?ip=${encodeURIComponent(ipAddress)}`, createAuthorizedOptions("DELETE")); blacklist.value = blacklist.value.filter((item) => item.ip !== ipAddress); metrics.blacklist_count = blacklist.value.length; showToast("已移出黑名单"); } catch (error) { showToast(error.message); } }
// applyReply 输入回复内容，输出为输入框填充结果，目的在于让客服一键使用快捷回复。
function applyReply(content) { composer.value = content; }
// formatAgentCapabilities 输入能力数组，输出为标签数组，目的在于给 Agent 面板提供兜底文案。
function formatAgentCapabilities(capabilities) { return Array.isArray(capabilities) && capabilities.length > 0 ? capabilities : ["基础接待"]; }
// handleComposerKeydown 输入键盘事件，输出为发送或换行结果，目的在于支持 Enter 发送和 Shift+Enter 换行。
function handleComposerKeydown(event) { if (event.key === "Enter" && !event.shiftKey) { event.preventDefault(); sendMessage(); } }
// logout 输入为空，输出为退出结果，目的在于清理本地凭证并返回登录页。
function logout() { clearAuthToken(); router.replace("/login"); }
// sendTypingEvent 输入为空，输出为输入中事件发送结果，目的在于向实时链路广播当前输入状态。
function sendTypingEvent() { if (!socketInstance || socketInstance.readyState !== WebSocket.OPEN || !currentVisitorId.value) return; const now = Date.now(); if (now - lastTypingAt < 800) return; lastTypingAt = now; socketInstance.send(JSON.stringify({ type: "inputing", data: { from: currentVisitorId.value, to: profile.username } })); }

watch(composer, () => sendTypingEvent());
onMounted(async () => { if (!getAuthToken()) { await router.replace("/login"); return; } try { await bootstrapWorkbench(); openSocket(); } catch (error) { showToast(error.message); } finally { isLoading.value = false; } });
onBeforeUnmount(() => { stopHeartbeat(); if (reconnectTimer) window.clearTimeout(reconnectTimer); if (socketInstance) socketInstance.close(); });
</script>
