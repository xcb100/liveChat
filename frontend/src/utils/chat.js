const originalDocumentTitle = document.title;
const faceTitles = ["[微笑]", "[嘻嘻]", "[哈哈]", "[可爱]", "[可怜]", "[挖鼻]", "[吃惊]", "[害羞]", "[挤眼]", "[闭嘴]", "[鄙视]", "[爱你]", "[泪]", "[偷笑]", "[亲亲]", "[生病]", "[太开心]", "[白眼]", "[右哼哼]", "[左哼哼]", "[嘘]", "[衰]", "[委屈]", "[吐]", "[哈欠]", "[抱抱]", "[怒]", "[疑问]", "[馋嘴]", "[拜拜]", "[思考]", "[汗]", "[困]", "[睡]", "[钱]", "[失望]", "[酷]", "[色]", "[哼]", "[鼓掌]", "[晕]", "[悲伤]", "[抓狂]", "[黑线]", "[阴险]", "[怒骂]", "[互粉]", "[心]", "[伤心]", "[猪头]", "[熊猫]", "[兔子]", "[ok]", "[耶]", "[good]", "[NO]", "[赞]", "[来]", "[弱]", "[草泥马]", "[神马]", "[囧]", "[浮云]", "[给力]", "[围观]", "[威武]", "[奥特曼]", "[礼物]", "[钟]", "[话筒]", "[蜡烛]", "[蛋糕]"];

let titleFlashTimer = 0;
let titleFlashIndex = 0;

// getWsBaseUrl 输入为空，输出为当前站点 WebSocket 地址，目的在于统一生成 ws 或 wss 协议地址。
export function getWsBaseUrl() {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}`;
}

// stripHtml 输入字符串，输出为去掉 HTML 标签的纯文本，目的在于避免桌面通知直接渲染消息 HTML。
export function stripHtml(content) {
  return String(content || "").replace(/<[^>]*>/g, "");
}

// notify 输入标题、选项和点击回调，输出为浏览器通知结果，目的在于在页面失焦时提醒用户有新消息。
export function notify(title, options = {}, callback = null) {
  if (!window.Notification) {
    return;
  }

  const normalizedOptions = {
    ...options,
    body: stripHtml(options.body || ""),
  };

  const openNotification = () => {
    const notification = new Notification(title, normalizedOptions);
    if (callback) {
      notification.onclick = (event) => callback(notification, event);
    }
    window.setTimeout(() => notification.close(), 3000);
  };

  if (Notification.permission === "granted") {
    openNotification();
    return;
  }

  if (Notification.permission === "default") {
    Notification.requestPermission().then((permission) => {
      if (permission === "granted") {
        openNotification();
      }
    });
  }
}

// flashTitle 输入为空，输出为标题闪烁状态，目的在于在新消息到达时提示失焦用户。
export function flashTitle() {
  if (titleFlashTimer !== 0) {
    return;
  }

  titleFlashTimer = window.setInterval(() => {
    titleFlashIndex += 1;
    if (titleFlashIndex >= 2) {
      titleFlashIndex = 0;
    }
    document.title = titleFlashIndex === 0 ? `【新消息】${originalDocumentTitle}` : `【     】${originalDocumentTitle}`;
  }, 600);
}

// clearFlashTitle 输入为空，输出为标题恢复结果，目的在于在用户回到页面后停止标题闪烁。
export function clearFlashTitle() {
  if (titleFlashTimer !== 0) {
    window.clearInterval(titleFlashTimer);
  }
  titleFlashTimer = 0;
  titleFlashIndex = 0;
  document.title = originalDocumentTitle;
}

// getFaceMap 输入为空，输出为表情映射，目的在于把表情占位文本转换成静态资源地址。
export function getFaceMap() {
  return faceTitles.reduce((faceMap, faceTitle, index) => {
    faceMap[faceTitle] = `/static/images/face/${index}.gif`;
    return faceMap;
  }, {});
}

// getFaceOptions 输入为空，输出为表情数据列表，目的在于为访客页表情面板提供标准数据源。
export function getFaceOptions() {
  const faceMap = getFaceMap();
  return faceTitles.map((faceTitle) => ({
    name: faceTitle,
    path: faceMap[faceTitle],
  }));
}

// formatFileSize 输入文件字节数，输出为可读大小文本，目的在于为附件卡片展示更友好的体积信息。
export function formatFileSize(fileSize) {
  const normalizedSize = Number(fileSize || 0);

  if (normalizedSize < 1024) {
    return `${normalizedSize}B`;
  }
  if (normalizedSize < 1024 * 1024) {
    return `${(normalizedSize / 1024).toFixed(2)}KB`;
  }
  if (normalizedSize < 1024 * 1024 * 1024) {
    return `${(normalizedSize / (1024 * 1024)).toFixed(2)}MB`;
  }
  return `${(normalizedSize / (1024 * 1024 * 1024)).toFixed(2)}GB`;
}

// replaceAttachment 输入消息文本，输出为带附件卡片的 HTML，目的在于兼容现有 attachment 协议展示。
export function replaceAttachment(content) {
  return String(content || "").replace(/attachment\[(.*?)\]/g, (matchedText) => {
    const matchedFiles = matchedText.match(/attachment\[(.*?)\]/);
    if (!matchedFiles || matchedFiles.length < 2) {
      return matchedText;
    }

    let attachmentInfo = null;
    try {
      attachmentInfo = JSON.parse(matchedFiles[1]);
    } catch (error) {
      return matchedText;
    }

    const extensionIconMap = {
      ".mp3": "/static/images/ext/MP3.png",
      ".zip": "/static/images/ext/ZIP.png",
      ".txt": "/static/images/ext/TXT.png",
      ".7z": "/static/images/ext/7z.png",
      ".bmp": "/static/images/ext/BMP.png",
      ".png": "/static/images/ext/PNG.png",
      ".jpg": "/static/images/ext/JPG.png",
      ".jpeg": "/static/images/ext/JPEG.png",
      ".pdf": "/static/images/ext/PDF.png",
      ".doc": "/static/images/ext/DOC.png",
      ".docx": "/static/images/ext/DOCX.png",
      ".rar": "/static/images/ext/RAR.png",
      ".xlsx": "/static/images/ext/XLSX.png",
      ".csv": "/static/images/ext/XLSX.png",
    };

    const iconPath = extensionIconMap[String(attachmentInfo.ext || "").toLowerCase()] || "/static/images/ext/default.png";

    return `
      <button class="chat-attachment-card" type="button" onclick="window.open('${attachmentInfo.path}', '_blank')">
        <img src="${iconPath}" alt="${attachmentInfo.ext || "file"}">
        <span class="chat-attachment-meta">
          <strong>${attachmentInfo.name || "附件"}</strong>
          <small>${formatFileSize(attachmentInfo.size)}</small>
        </span>
      </button>
    `;
  });
}

// renderChatContent 输入消息文本，输出为渲染后的 HTML 字符串，目的在于统一处理表情、图片、附件和换行协议。
export function renderChatContent(content) {
  const faceMap = getFaceMap();

  const renderedContent = String(content || "")
    .replace(/face(\[[^\]]+\])/g, (matchedText, faceTitle) => {
      const facePath = faceMap[faceTitle];
      if (!facePath) {
        return matchedText;
      }
      return `<img class="chat-inline-face" alt="${faceTitle}" title="${faceTitle}" src="${facePath}">`;
    })
    .replace(/img\[(.*?)\]/g, (matchedText, imageSource) => {
      return `<img class="chat-inline-image" src="${imageSource}" alt="chat image" onclick="window.open('${imageSource}', '_blank')">`;
    })
    .replace(/\n/g, "<br>");

  return replaceAttachment(renderedContent);
}

// formatDate 输入日期字符串和格式串，输出为格式化时间文本，目的在于在聊天界面统一展示时间。
export function formatDate(dateString, format = "yyyy-MM-dd HH:mm:ss") {
  const dateObject = new Date(dateString);
  if (Number.isNaN(dateObject.getTime())) {
    return "";
  }

  const formattedValues = {
    yyyy: String(dateObject.getFullYear()),
    MM: String(dateObject.getMonth() + 1).padStart(2, "0"),
    dd: String(dateObject.getDate()).padStart(2, "0"),
    HH: String(dateObject.getHours()).padStart(2, "0"),
    mm: String(dateObject.getMinutes()).padStart(2, "0"),
    ss: String(dateObject.getSeconds()).padStart(2, "0"),
  };

  return Object.entries(formattedValues).reduce((currentFormat, [token, tokenValue]) => currentFormat.replace(token, tokenValue), format);
}

// isSupportedImageFile 输入文件对象，输出为是否支持的图片格式，目的在于过滤聊天图片上传类型。
export function isSupportedImageFile(file) {
  const supportedTypes = ["image/jpeg", "image/png", "image/jpg", "image/gif"];
  return supportedTypes.includes(file?.type || "");
}

// utf8ToBase64 输入 UTF-8 文本，输出为 Base64 文本，目的在于兼容访客扩展信息透传协议。
export function utf8ToBase64(content) {
  return window.btoa(unescape(encodeURIComponent(content)));
}

// base64ToUtf8 输入 Base64 文本，输出为 UTF-8 文本，目的在于解析访客扩展资料和旧字段编码。
export function base64ToUtf8(content) {
  return decodeURIComponent(escape(window.atob(content)));
}

// playAudio 输入音频元素标识和资源地址，输出为播放结果，目的在于为收发消息提供统一提示音入口。
export function playAudio(elementId, sourcePath = "") {
  const audioElement = document.getElementById(elementId);
  if (!audioElement) {
    return;
  }
  if (sourcePath) {
    audioElement.src = sourcePath;
  }
  const playResult = audioElement.play();
  if (playResult && typeof playResult.catch === "function") {
    playResult.catch(() => {});
  }
}
