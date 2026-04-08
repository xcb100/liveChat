<template>
  <div class="app-page visitor-page">
    <div class="livechat-shell">
      <main class="visitor-panel">
        <header v-if="!isIframe" class="visitor-header">
          <div class="visitor-brand">
            <img class="visitor-avatar is-large" :src="kefuInfo.avatar || '/static/images/admin.png'" alt="agent avatar">
            <div>
              <h1 class="visitor-title">在线客服</h1>
              <p class="visitor-subtitle">{{ kefuInfo.nickname || '在线客服' }} 正在为你服务</p>
            </div>
          </div>
          <div class="status-chip" :class="connectionChipClass"><span class="status-dot"></span>{{ statusText }}</div>
        </header>

        <InlineAlert :message="toastMessage" variant="info" />

        <section ref="messageAreaRef" class="visitor-message-area">
          <button v-if="showLoadMore" class="secondary-button load-more-button" type="button" @click="loadMoreMessages">加载更早消息</button>
          <div class="visitor-message-list">
            <article v-for="message in messages" :key="message.key" class="visitor-message-entry">
              <div v-if="message.show_time" class="visitor-message-time"><span>{{ message.time }}</span></div>
              <div class="visitor-message-item" :class="{ 'is-self': !message.isAgent }">
                <img class="chat-avatar" :src="message.avator" alt="message avatar">
                <div class="message-body">
                  <div class="message-name">{{ message.name }}</div>
                  <div class="message-bubble" v-html="message.content"></div>
                </div>
              </div>
            </article>
          </div>
        </section>

        <section class="visitor-composer">
          <div class="visitor-toolbar">
            <div class="visitor-actions">
              <button class="icon-button" type="button" @click.stop="showFacePanel = !showFacePanel">笑</button>
              <button class="icon-button" type="button" @click="triggerFileInput('image')">图</button>
              <button class="icon-button" type="button" @click="triggerFileInput('file')">文</button>
            </div>
            <div class="visitor-muted">{{ isUploading ? '上传中，请稍候' : '支持图片、附件、表情和截图粘贴' }}</div>
          </div>

          <div v-if="showFacePanel" ref="faceBoxRef" class="workspace-card" style="margin-bottom: 12px;">
            <div class="visitor-face-grid">
              <button v-for="(faceItem, faceIndex) in faceOptions" :key="faceItem.name" class="visitor-face-button" type="button" @click="appendFace(faceIndex)">
                <img :src="faceItem.path" :alt="faceItem.name">
              </button>
            </div>
          </div>

          <textarea class="panel-textarea" v-model="messageContent" maxlength="100" placeholder="输入消息内容，按 Enter 发送，Shift + Enter 换行" @keydown="handleComposerKeydown"></textarea>
          <div class="visitor-actions" style="margin-top: 12px;">
            <button class="secondary-button" type="button" @click="messageContent = ''">清空</button>
            <button class="primary-button" :disabled="sendDisabled || !messageContent.trim()" type="button" @click="sendMessage">发送</button>
          </div>
        </section>

        <input ref="imageInputRef" type="file" accept="image/gif,image/jpeg,image/jpg,image/png" hidden @change="handleImageUpload">
        <input ref="fileInputRef" type="file" hidden @change="handleFileUpload">
      </main>

      <aside class="visitor-side-panel">
        <section class="visitor-card">
          <div class="visitor-card-header">
            <h2 class="visitor-title" style="font-size: 22px;">当前接待</h2>
          </div>
          <div class="visitor-notice">
            <strong>{{ kefuInfo.nickname || '在线客服' }}</strong>
            <div class="visitor-muted" style="margin-top: 8px;">当前会话连接到 {{ visitor.to_id || kefuId || '默认客服' }}。</div>
          </div>
        </section>
        <section class="visitor-card">
          <div class="visitor-card-header">
            <h2 class="visitor-title" style="font-size: 22px;">公告</h2>
          </div>
          <div class="visitor-notice" v-html="renderedNotice || '当前暂无公告。'"></div>
        </section>
        <section class="visitor-card">
          <div class="visitor-card-header">
            <h2 class="visitor-title" style="font-size: 22px;">聊天说明</h2>
          </div>
          <div class="visitor-notice">
            <div>支持图片、附件、表情和截图粘贴发送。</div>
            <div>若连接意外中断，页面会自动尝试重连。</div>
            <div>消息发送频率受到限流保护，避免刷屏与误触。</div>
          </div>
        </section>
      </aside>

      <audio id="chatMessageAudio"></audio>
      <audio id="chatMessageSendAudio"></audio>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { useRoute } from "vue-router";
import InlineAlert from "@/components/InlineAlert.vue";
import { createFormData, requestJson } from "@/services/http";
import { buildVisitorCacheKey, loadVisitorCache, saveVisitorCache } from "@/services/session";
import { clearFlashTitle, flashTitle, formatDate, getFaceOptions, getWsBaseUrl, isSupportedImageFile, notify, playAudio, renderChatContent, utf8ToBase64 } from "@/utils/chat";

const route = useRoute();
const kefuId = computed(() => String(route.query.user_id || ""));
const isIframe = window.self !== window.top;
const messageAreaRef = ref(null);
const imageInputRef = ref(null);
const fileInputRef = ref(null);
const faceBoxRef = ref(null);
const socketState = ref("connecting");
const statusText = ref("正在连接客服");
const toastMessage = ref("");
const messageContent = ref("");
const showFacePanel = ref(false);
const showLoadMore = ref(false);
const sendDisabled = ref(false);
const isUploading = ref(false);
const faceOptions = ref(getFaceOptions());
const messages = ref([]);
const visitor = reactive({ visitor_id: "", to_id: "", name: "", avator: "" });
const kefuInfo = reactive({ nickname: "在线客服", avatar: "/static/images/admin.png", allNotice: "", welcome: "" });
const historyState = reactive({ page: 1, pagesize: 5 });
let socketInstance = null;
let heartbeatTimer = 0;
let reconnectTimer = 0;
let socketClosed = false;
let shouldReconnect = true;

const connectionChipClass = computed(() => socketState.value === "online" ? "" : socketState.value === "connecting" ? "is-warning" : "is-danger");
const renderedNotice = computed(() => renderChatContent(kefuInfo.allNotice || ""));

// showToast 输入文本，输出为提示状态，目的在于统一反馈上传、发送和连接错误。
function showToast(messageText) { toastMessage.value = messageText; window.setTimeout(() => { if (toastMessage.value === messageText) toastMessage.value = ""; }, 2400); }
// buildVisitorExtra 输入为空，输出为访客扩展编码，目的在于把 URL 中的访客信息透传给后端 visitor_login 接口。
function buildVisitorExtra() { const visitorName = String(route.query.name || route.query.visitor_name || ""); const visitorAvatar = String(route.query.avatar || route.query.visitor_avatar || ""); if (!visitorName && !visitorAvatar) return ""; return utf8ToBase64(JSON.stringify({ visitorName, visitorAvatar })); }
// normalizeHistoryMessage 输入原始消息，输出为统一消息结构，目的在于兼容历史记录与实时消息共用渲染层。
function normalizeHistoryMessage(messageRecord) { const rawFlag = String(messageRecord.is_kefu ?? messageRecord.mes_type ?? "").toLowerCase(); const isAgent = rawFlag === "yes" || rawFlag === "true" || rawFlag === "kefu"; return { key: `${messageRecord.id || messageRecord.create_time || Date.now()}-${Math.random()}`, time: messageRecord.create_time || messageRecord.time || formatDate(new Date().toISOString()), content: renderChatContent(messageRecord.content || ""), isAgent, name: isAgent ? messageRecord.kefu_name || messageRecord.name || "客服" : messageRecord.visitor_name || visitor.name || "访客", avator: isAgent ? messageRecord.kefu_avator || messageRecord.avator || "/static/images/admin.png" : messageRecord.visitor_avator || messageRecord.avator || visitor.avator || "/static/images/2.png", show_time: messageRecord.show_time !== false }; }
// buildNoticeMessage 输入提示文本，输出为系统提示消息对象，目的在于统一会话状态提示展示。
function buildNoticeMessage(noticeText) { return { key: `notice-${Date.now()}-${Math.random()}`, time: formatDate(new Date().toISOString()), content: noticeText, isAgent: true, name: "系统", avator: "/static/images/admin.png", show_time: true }; }
// scrollToBottom 输入为空，输出为滚动结果，目的在于保持消息区域始终聚焦最新消息。
async function scrollToBottom() { await nextTick(); if (messageAreaRef.value) messageAreaRef.value.scrollTop = messageAreaRef.value.scrollHeight; }
// initializeVisitor 输入为空，输出为访客初始化结果，目的在于复用 visitor_id、完成登录并拉取首屏数据。
async function initializeVisitor() { if (!kefuId.value) throw new Error("缺少客服标识 user_id"); const cacheKey = buildVisitorCacheKey(kefuId.value); const cachedVisitor = loadVisitorCache(cacheKey); const visitorInfo = await requestJson("/visitor_login", { method: "POST", body: createFormData({ visitor_id: cachedVisitor?.visitor_id || route.query.visitor_id || "", refer: document.referrer || "Direct access", to_id: kefuId.value, extra: buildVisitorExtra() }) }); Object.assign(visitor, visitorInfo || {}); saveVisitorCache(cacheKey, visitorInfo); await loadHistoryMessages(true); await loadNotice(); openSocket(); }
// loadHistoryMessages 输入是否重置，输出为历史消息结果，目的在于分页读取访客与客服的聊天记录。
async function loadHistoryMessages(reset) { if (!visitor.visitor_id) return; if (reset) { historyState.page = 1; messages.value = []; } const result = await requestJson(`/2/messagesPages?visitor_id=${encodeURIComponent(visitor.visitor_id)}&page=${historyState.page}&pagesize=${historyState.pagesize}`); const historyMessages = (result.list || []).slice().reverse().map((item) => normalizeHistoryMessage(item)); messages.value = reset ? historyMessages : [...historyMessages, ...messages.value]; showLoadMore.value = messages.value.length < (result.count || 0); historyState.page += 1; if (reset) await scrollToBottom(); }
// loadMoreMessages 输入为空，输出为更多历史加载结果，目的在于让访客继续向上翻阅更早消息。
async function loadMoreMessages() { await loadHistoryMessages(false); }
// loadNotice 输入为空，输出为公告和欢迎语结果，目的在于首屏展示客服公告和欢迎消息。
async function loadNotice() { const result = await requestJson(`/notice?kefu_id=${encodeURIComponent(kefuId.value)}`); Object.assign(kefuInfo, result || {}); statusText.value = `${kefuInfo.nickname || '在线客服'} 为你服务`; if (kefuInfo.welcome) { messages.value.push({ key: `welcome-${Date.now()}`, time: formatDate(new Date().toISOString()), content: renderChatContent(kefuInfo.welcome), isAgent: true, name: kefuInfo.nickname || '客服', avator: kefuInfo.avatar || '/static/images/admin.png', show_time: true }); playAudio('chatMessageAudio', '/static/images/alert2.ogg'); await scrollToBottom(); } }
// startHeartbeat 输入为空，输出为心跳启动结果，目的在于维持访客侧 WebSocket 活跃状态。
function startHeartbeat() { stopHeartbeat(); heartbeatTimer = window.setInterval(() => { if (socketInstance?.readyState === WebSocket.OPEN) socketInstance.send(JSON.stringify({ type: 'ping', data: `visitor:${visitor.visitor_id}` })); }, 10000); }
// stopHeartbeat 输入为空，输出为心跳停止结果，目的在于避免重复心跳定时器泄漏。
function stopHeartbeat() { if (heartbeatTimer) { window.clearInterval(heartbeatTimer); heartbeatTimer = 0; } }
// scheduleReconnect 输入为空，输出为重连调度结果，目的在于连接异常中断后自动恢复会话。
function scheduleReconnect() { if (reconnectTimer || socketClosed) return; reconnectTimer = window.setTimeout(() => { reconnectTimer = 0; openSocket(); }, 3000); }
// openSocket 输入为空，输出为 WebSocket 建连结果，目的在于建立访客实时会话通道。
function openSocket() { if (!visitor.visitor_id) return; stopHeartbeat(); if (reconnectTimer) { window.clearTimeout(reconnectTimer); reconnectTimer = 0; } socketState.value = 'connecting'; statusText.value = '正在连接客服'; socketInstance = new WebSocket(`${getWsBaseUrl()}/ws_visitor?visitor_id=${encodeURIComponent(visitor.visitor_id)}`); socketInstance.onopen = () => { socketState.value = 'online'; socketClosed = false; statusText.value = '连接正常'; startHeartbeat(); }; socketInstance.onmessage = (event) => { handleSocketMessage(event.data); }; socketInstance.onerror = () => { socketState.value = 'offline'; statusText.value = '连接异常，正在重试'; }; socketInstance.onclose = () => { stopHeartbeat(); if (!shouldReconnect) return; socketState.value = socketClosed ? 'offline' : 'connecting'; if (!socketClosed) { statusText.value = '连接断开，准备重连'; scheduleReconnect(); } }; }
// handleSocketMessage 输入消息文本，输出为状态更新结果，目的在于统一消费访客侧实时事件。
async function handleSocketMessage(rawMessage) { const payload = JSON.parse(rawMessage); if (payload.type === 'pong') return; if (payload.type === 'notice') { if (payload.data) messages.value.push(buildNoticeMessage(payload.data)); await scrollToBottom(); return; } if (payload.type === 'message') { const rawFlag = String(payload.data?.is_kefu ?? payload.data?.mes_type ?? '').toLowerCase(); const isAgent = rawFlag === 'yes' || rawFlag === 'true' || rawFlag === 'kefu'; const normalizedMessage = { key: `${payload.data?.id || Date.now()}-${Math.random()}`, time: payload.data?.time || formatDate(new Date().toISOString()), content: renderChatContent(payload.data?.content || ''), isAgent, name: payload.data?.name || (isAgent ? '客服' : visitor.name || '访客'), avator: payload.data?.avator || (isAgent ? '/static/images/admin.png' : visitor.avator || '/static/images/2.png'), show_time: true }; messages.value.push(normalizedMessage); if (isIframe) window.parent.postMessage({ type: 'new_message' }, '*'); notify(payload.data?.name || '新消息', { body: payload.data?.content || '', icon: payload.data?.avator || kefuInfo.avatar }, () => window.focus()); flashTitle(); playAudio('chatMessageAudio', '/static/images/alert2.ogg'); await scrollToBottom(); return; } if (['close', 'force_close', 'auto_close'].includes(payload.type)) { socketClosed = true; shouldReconnect = false; socketState.value = 'offline'; statusText.value = payload.type === 'force_close' ? '客服已结束会话' : payload.type === 'auto_close' ? '会话因长时间无操作而结束' : '当前会话已结束'; messages.value.push(buildNoticeMessage(statusText.value)); if (isIframe) window.parent.postMessage({ type: 'close_chat' }, '*'); socketInstance?.close(); } }
// sendMessage 输入为空，输出为发送结果，目的在于把当前输入内容发送给客服端。
async function sendMessage() { const content = messageContent.value.trim(); if (!content) { messageContent.value = ''; return; } if (socketClosed) { showToast('连接已关闭，请刷新页面后重试'); return; } sendDisabled.value = true; const optimisticMessage = { key: `send-${Date.now()}`, time: formatDate(new Date().toISOString()), content: renderChatContent(content), isAgent: false, name: visitor.name || '访客', avator: visitor.avator || '/static/images/2.png', show_time: false }; messages.value.push(optimisticMessage); await scrollToBottom(); try { await requestJson('/2/message', { method: 'POST', body: createFormData({ type: 'visitor', content, from_id: visitor.visitor_id, to_id: visitor.to_id }) }); messageContent.value = ''; playAudio('chatMessageSendAudio', '/static/images/sent.ogg'); } catch (error) { messages.value = messages.value.filter((item) => item.key !== optimisticMessage.key); showToast(error.message); } finally { sendDisabled.value = false; } }
// handleComposerKeydown 输入键盘事件，输出为发送或换行结果，目的在于支持 Enter 发送和 Shift+Enter 换行。
function handleComposerKeydown(event) { if (event.key === 'Enter' && !event.shiftKey) { event.preventDefault(); sendMessage(); } }
// appendFace 输入表情序号，输出为输入框追加结果，目的在于向当前消息插入表情占位文本。
function appendFace(index) { if (!faceOptions.value[index]) return; messageContent.value += `face${faceOptions.value[index].name}`; showFacePanel.value = false; }
// triggerFileInput 输入类型标识，输出为文件选择动作，目的在于复用隐藏上传控件。
function triggerFileInput(type) { const inputMap = { image: imageInputRef.value, file: fileInputRef.value }; inputMap[type]?.click(); }
// uploadImageAsset 输入图片文件，输出为上传结果，目的在于为图片消息和粘贴截图复用同一上传逻辑。
async function uploadImageAsset(file) { const formData = new FormData(); formData.append('imgfile', file); return requestJson('/uploadimg', { method: 'POST', body: formData }); }
// handleImageUpload 输入文件事件，输出为图片消息发送结果，目的在于上传图片后插入标准协议发送。
async function handleImageUpload(event) { const file = event.target.files?.[0]; event.target.value = ''; if (!file) return; if (!isSupportedImageFile(file)) { showToast('仅支持 png、jpg、jpeg、gif 图片'); return; } isUploading.value = true; try { const result = await uploadImageAsset(file); messageContent.value += `img[/${result.path}]`; await sendMessage(); } catch (error) { showToast(error.message); } finally { isUploading.value = false; } }
// handleFileUpload 输入文件事件，输出为附件消息发送结果，目的在于上传附件后插入标准附件协议。
async function handleFileUpload(event) { const file = event.target.files?.[0]; event.target.value = ''; if (!file) return; isUploading.value = true; try { const formData = new FormData(); formData.append('realfile', file); const result = await requestJson('/uploadfile', { method: 'POST', body: formData }); messageContent.value += `attachment[${JSON.stringify({ name: result.name, ext: result.ext, size: result.size, path: `/${result.path}` })}]`; await sendMessage(); } catch (error) { showToast(error.message); } finally { isUploading.value = false; } }
// handlePasteUpload 输入粘贴事件，输出为截图上传结果，目的在于支持用户直接粘贴图片发送。
async function handlePasteUpload(event) { const items = event.clipboardData?.items || []; const pastedImage = [...items].find((item) => item.type.includes('image'))?.getAsFile(); if (!pastedImage) return; isUploading.value = true; try { const result = await uploadImageAsset(pastedImage); messageContent.value += `img[/${result.path}]`; await sendMessage(); } catch (error) { showToast(error.message); } finally { isUploading.value = false; } }
// handleWindowFocus 输入为空，输出为焦点恢复结果，目的在于在用户回到页面后清理标题闪烁并必要时重连。
function handleWindowFocus() { clearFlashTitle(); if (!socketClosed && socketState.value !== 'online') { shouldReconnect = true; openSocket(); } }
// handleDocumentClick 输入点击事件，输出为面板收起结果，目的在于点击空白区域时关闭表情面板。
function handleDocumentClick(event) { if (showFacePanel.value && faceBoxRef.value && !faceBoxRef.value.contains(event.target)) showFacePanel.value = false; }

onMounted(async () => { document.addEventListener('paste', handlePasteUpload); document.addEventListener('click', handleDocumentClick); window.addEventListener('focus', handleWindowFocus); try { await initializeVisitor(); } catch (error) { socketClosed = true; shouldReconnect = false; socketState.value = 'offline'; statusText.value = error.message; showToast(error.message); } });
onBeforeUnmount(() => { shouldReconnect = false; stopHeartbeat(); if (reconnectTimer) window.clearTimeout(reconnectTimer); document.removeEventListener('paste', handlePasteUpload); document.removeEventListener('click', handleDocumentClick); window.removeEventListener('focus', handleWindowFocus); if (socketInstance) socketInstance.close(); });
</script>
