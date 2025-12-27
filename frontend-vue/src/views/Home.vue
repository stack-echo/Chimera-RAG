<template>
  <div class="app-container">
    <div class="header">
      <div class="brand">🦄 Chimera-RAG</div>
      <div class="user-info">
        <span>{{ userStore.userInfo.username }}</span>
        <a-button type="text" status="danger" size="mini" @click="handleLogout">退出</a-button>
      </div>
    </div>

    <div class="main-content">
      <div class="chat-panel" :class="{ 'full-width': !currentPdfUrl }">
        <div class="messages" ref="msgListRef">
          <div v-for="(msg, index) in messages" :key="index" :class="['message', msg.role]">
            <div class="avatar">{{ msg.role === 'user' ? '👤' : '🤖' }}</div>
            <div class="content">
              <div v-if="msg.thinking" class="thinking-box">
                <div class="think-title">Thinking...</div>
                <div class="think-content">{{ msg.thinking }}</div>
              </div>

              <div v-html="renderMarkdown(msg.content)"></div>

              <div v-if="msg.citations && msg.citations.length" class="citation-box">
                <div class="citation-title">参考来源:</div>
                <div
                    v-for="(cite, idx) in msg.citations"
                    :key="idx"
                    class="citation-item"
                    @click="openPdfPage(cite.file_name, cite.page_number)"
                >
                  📄 {{ cite.file_name }} (P{{ cite.page_number }})
                </div>
              </div>
            </div>
          </div>
          <div v-if="loading" class="loading">AI 正在思考...</div>
        </div>

        <div class="input-area">
          <a-upload action="/" :custom-request="customRequest" :show-file-list="false">
            <template #upload-button>
              <a-button type="secondary" shape="circle"><icon-upload /></a-button>
            </template>
          </a-upload>
          <a-input v-model="inputVal" @press-enter="sendMsg" placeholder="输入问题..." style="margin: 0 10px; flex: 1" />
          <a-button type="primary" @click="sendMsg" :disabled="loading">发送</a-button>
        </div>
      </div>

      <div class="pdf-panel" v-if="currentPdfUrl">
        <div class="pdf-header">
          <span class="pdf-title">📄 {{ currentPdfName }}</span>
          <a-button size="mini" @click="closePdf">关闭</a-button>
        </div>
        <div class="pdf-viewer" ref="pdfContainer">
          <VuePdfEmbed
              :source="currentPdfUrl"
              :page="targetPage"
              class="pdf-embed"
              width="800"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
// 1. 所有的 Import 必须放在顶部
import { ref, nextTick, onMounted, onUnmounted } from 'vue'
import VuePdfEmbed from 'vue-pdf-embed'
import MarkdownIt from 'markdown-it'
import { IconUpload } from '@arco-design/web-vue/es/icon'
import request from '../api/request'
import { useUserStore } from '../store/user'
import { useRouter } from 'vue-router'
import { fetchEventSource } from '@microsoft/fetch-event-source'

const userStore = useUserStore()
const router = useRouter()
const md = new MarkdownIt()

// 状态
const messages = ref([])
const inputVal = ref('')
const loading = ref(false)
const msgListRef = ref(null)

// PDF 预览状态
const currentPdfUrl = ref('')
const currentPdfName = ref('')
const targetPage = ref(1) // 控制显示的页码，如果不传则显示全部

// ---------------------------------------------------------
// 🛠️ 事件监听处理 (修复 onUnmounted 报错)
// ---------------------------------------------------------

// 定义一个具名函数，方便 add 和 remove
const handleOpenPdfEvent = (e) => {
  if (e.detail) {
    console.log('接收到跳转事件:', e.detail)
    openPdfPage(e.detail.filename, parseInt(e.detail.page))
  }
}

// 挂载全局方法给 HTML 字符串里的 onclick 调用
window.openPdf = (filename, page) => {
  const event = new CustomEvent('open-pdf', { detail: { filename, page } });
  window.dispatchEvent(event);
}

onMounted(() => {
  window.addEventListener('open-pdf', handleOpenPdfEvent)
})

onUnmounted(() => {
  // 🔥 修复点：必须传入同一个函数引用，且不能写 ...
  window.removeEventListener('open-pdf', handleOpenPdfEvent)
  // 清理 Blob URL 避免内存泄漏
  if (currentPdfUrl.value) URL.revokeObjectURL(currentPdfUrl.value)
})

// ---------------------------------------------------------
// 业务逻辑
// ---------------------------------------------------------

const handleLogout = () => {
  userStore.logout()
  router.push('/login')
}

// 渲染 MD
const renderMarkdown = (text) => {
  if (!text) return ''
  let html = md.render(text)

  // 替换引用格式 <<filename|page>>
  const citationRegex = /(&lt;&lt;|<<)\s*(.*?)\s*\|\s*(\d+)\s*(&gt;&gt;|>>)/g;
  html = html.replace(citationRegex, (match, p1, filename, page) => {
    return `<span class="citation-highlight" onclick="window.openPdf('${filename}', ${page})">📄 [P${page}]</span>`
  })
  return html
}

// 上传
const customRequest = async (option) => {
  const { onError, onSuccess, fileItem } = option
  const formData = new FormData()
  formData.append('file', fileItem.file)

  try {
    const res = await request.post('/upload', formData)
    onSuccess(res)
    // 假设后端返回 res.path 是文件名
    openPdfPage(res.path, 1)
    messages.value.push({ role: 'assistant', content: `✅ 文件 **${fileItem.file.name}** 上传成功！正在后台解析...` })
  } catch (error) {
    onError(error)
  }
}

// 打开 PDF (获取 Blob)
const openPdfPage = (filename, page) => {
  // 如果已经在看这个文件，只跳页码
  if (currentPdfName.value === filename && currentPdfUrl.value) {
    targetPage.value = page
    return
  }
  fetchPdfBlob(filename, page)
}

const fetchPdfBlob = async (filename, page) => {
  try {
    const res = await request.get(`/file/${filename}`, { responseType: 'blob' })
    const blob = new Blob([res], { type: 'application/pdf' })

    // 释放旧的 URL
    if (currentPdfUrl.value) URL.revokeObjectURL(currentPdfUrl.value)

    currentPdfUrl.value = URL.createObjectURL(blob)
    currentPdfName.value = filename
    targetPage.value = page
  } catch (e) {
    console.error("加载PDF失败", e)
  }
}

const closePdf = () => {
  if (currentPdfUrl.value) URL.revokeObjectURL(currentPdfUrl.value)
  currentPdfUrl.value = ''
  currentPdfName.value = ''
}

// 发送消息
const sendMsg = async () => {
  if (!inputVal.value.trim()) return
  messages.value.push({ role: 'user', content: inputVal.value })
  const userQ = inputVal.value
  inputVal.value = ''
  loading.value = true

  const aiMsgIndex = messages.value.length
  messages.value.push({ role: 'assistant', content: '', thinking: '', citations: [] })

  try {
    await fetchEventSource('http://localhost:8080/api/v1/chat/stream', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${userStore.token}`
      },
      body: JSON.stringify({ query: userQ }),
      onmessage(msg) {
        const data = msg.data
        if (data.startsWith('THINKing: ')) {
          messages.value[aiMsgIndex].thinking += data.replace('THINKing: ', '') + '\n'
        } else if (data.startsWith('ANSWER: ')) {
          // 兼容 v0.2.0/v0.3.0 的后端逻辑，如果后端发的是 ANSWER: 前缀
          messages.value[aiMsgIndex].content += data.replace('ANSWER: ', '')
        } else if (!data.startsWith('SOURCE: ')) {
          // 默认处理 (假设全是正文)
          messages.value[aiMsgIndex].content += data
        }

        nextTick(() => {
          if(msgListRef.value) msgListRef.value.scrollTop = msgListRef.value.scrollHeight
        })
      },
      onclose() { loading.value = false },
      onerror(err) { throw err }
    })
  } catch (err) {
    loading.value = false
    messages.value[aiMsgIndex].content += '\n*(网络连接异常)*'
  }
}
</script>

<style scoped>
.app-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #f5f7fa;
}
.header {
  height: 50px;
  background: white;
  border-bottom: 1px solid #ddd;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 20px;
}
.brand { font-weight: bold; font-size: 18px; }
.main-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* 左侧聊天 */
.chat-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  border-right: 1px solid #ddd;
  max-width: 50%; /* 默认宽度 */
  transition: max-width 0.3s ease; /* 加个动画 */
}
/* 🔥 关键优化：如果没有 PDF，聊天框占满 */
.chat-panel.full-width {
  max-width: 100%;
  border-right: none;
}

.messages { flex: 1; overflow-y: auto; padding: 20px; }
.input-area { padding: 20px; background: white; display: flex; align-items: center; }

/* 右侧 PDF */
.pdf-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #525659;
  min-width: 0;
  height: 100%;
}

.pdf-header {
  height: 40px;
  background: #333;
  color: white;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 15px;
}
.pdf-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 80%;
}

.pdf-viewer {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  justify-content: center;
}
.pdf-embed {
  box-shadow: 0 4px 10px rgba(0,0,0,0.3);
  /* 确保 PDF 不会撑破容器，并在容器内自适应 */
  width: 90%;
  height: auto;
  display: block;
}

/* 样式穿透 */
:deep(.citation-highlight) {
  color: #165dff;
  font-weight: bold;
  cursor: pointer;
  background: rgba(22, 93, 255, 0.1);
  padding: 2px 6px;
  border-radius: 4px;
  margin: 0 2px;
}
:deep(.citation-highlight:hover) {
  background: rgba(22, 93, 255, 0.2);
  text-decoration: underline;
}
.think-content {
  white-space: pre-wrap;
  font-family: monospace;
}
/* 复用消息样式 */
.message { display: flex; margin-bottom: 20px; }
.message.user { flex-direction: row-reverse; }
.content { background: white; padding: 10px; border-radius: 8px; max-width: 80%; }
.thinking-box { background: #f0f9ff; padding: 8px; font-size: 0.85em; color: #666; border-left: 3px solid #165dff; margin-bottom: 5px; }
</style>