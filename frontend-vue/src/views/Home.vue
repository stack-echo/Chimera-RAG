<template>
  <div class="chat-container">
    <div class="header">
      <h2>🦄 Chimera-RAG Knowledge Base</h2>
      <a-button type="text" status="danger" @click="handleLogout">
        退出登录 ({{ userStore.userInfo.username }})
      </a-button>
    </div>

    <div class="messages" ref="msgListRef">
      <div v-for="(msg, index) in messages" :key="index" :class="['message', msg.role]">
        <div class="avatar">{{ msg.role === 'user' ? '👤' : '🤖' }}</div>
        <div class="content">
          <div v-if="msg.thinking" class="thinking-box">
            <div class="think-title">Thinking Process...</div>
            <div class="think-content">{{ msg.thinking }}</div>
          </div>
          <div v-html="renderMarkdown(msg.content)"></div>
        </div>
      </div>
      <div v-if="loading" class="loading">AI 正在思考...</div>
    </div>

    <div class="input-area">
      <a-upload
        action="/"
        :custom-request="customRequest"
        :show-file-list="false"
      >
        <template #upload-button>
          <a-button type="outline" shape="circle"><icon-upload /></a-button>
        </template>
      </a-upload>

      <a-input
        v-model="inputVal"
        @press-enter="sendMsg"
        placeholder="输入问题，按回车发送..."
        style="margin: 0 10px; flex: 1"
      />
      <a-button type="primary" @click="sendMsg" :disabled="loading">发送</a-button>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick } from 'vue'
import { fetchEventSource } from '@microsoft/fetch-event-source'
import MarkdownIt from 'markdown-it'
import { IconUpload } from '@arco-design/web-vue/es/icon'
import request from '../api/request' // 使用封装的 axios
import { useUserStore } from '../store/user'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'

const userStore = useUserStore()
const router = useRouter()
const md = new MarkdownIt()

const messages = ref([])
const inputVal = ref('')
const loading = ref(false)
const msgListRef = ref(null)

// 渲染 Markdown
const renderMarkdown = (text) => {
  return md.render(text || '')
}

// 退出登录
const handleLogout = () => {
  userStore.logout()
  router.push('/login')
  Message.success('已退出')
}

// 📤 上传文件 (改造版)
const customRequest = async (option) => {
  const { onProgress, onError, onSuccess, fileItem, name } = option
  const formData = new FormData()
  formData.append(name || 'file', fileItem.file)

  try {
    // request 拦截器会自动带上 Token
    const res = await request.post('/upload', formData, {
      onUploadProgress: (event) => {
        let percent
        if (event.total > 0) {
          percent = (event.loaded / event.total) * 100
        }
        onProgress(percent, event)
      }
    })
    Message.success('上传成功')
    onSuccess(res)

    // 把上传结果作为一条系统消息展示
    messages.value.push({
      role: 'assistant',
      content: `📄 文件 **${fileItem.file.name}** 上传成功！(DocID: ${res.doc_id})`
    })
  } catch (error) {
    onError(error)
  }
}

// 💬 发送消息 (改造版 SSE)
const sendMsg = async () => {
  if (!inputVal.value.trim()) return

  // 添加用户消息
  messages.value.push({ role: 'user', content: inputVal.value })
  const userQ = inputVal.value
  inputVal.value = ''
  loading.value = true

  // 添加 AI 占位消息
  const aiMsgIndex = messages.value.length
  messages.value.push({ role: 'assistant', content: '', thinking: '' })

  try {
    await fetchEventSource('http://localhost:8080/api/v1/chat/stream', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        // 🔥 必须手动添加 Authorization 头，因为 fetchEventSource 不走 axios 拦截器
        'Authorization': `Bearer ${userStore.token}`
      },
      body: JSON.stringify({ query: userQ }),

      onmessage(msg) {
        // 1. 处理思考过程
          if (msg.data.startsWith('THINKing: ')) {
             messages.value[aiMsgIndex].thinking += msg.data.replace('THINKing: ', '') + '\n'
          }
          // 2. 处理错误
          else if (msg.data.startsWith('ERR: ')) {
             messages.value[aiMsgIndex].content += '\n**Error:** ' + msg.data
          }
          // 3. 🔥 修复点：处理正文
          // 如果后端发来的数据带有 "ANSWER: " 前缀，需要 strip 掉
          else {
             let cleanText = msg.data;
             if (cleanText.startsWith('ANSWER: ')) {
                 cleanText = cleanText.replace('ANSWER: ', '');
             }
             messages.value[aiMsgIndex].content += cleanText;
          }
        // 滚动到底部
        nextTick(() => {
          if (msgListRef.value) {
             msgListRef.value.scrollTop = msgListRef.value.scrollHeight
          }
        })
      },
      onclose() {
        loading.value = false
      },
      onerror(err) {
        console.error(err)
        loading.value = false
        throw err // rethrow to stop
      }
    })
  } catch (err) {
    loading.value = false
    messages.value[aiMsgIndex].content += '\n*(连接中断)*'
  }
}
</script>

<style scoped>
/* 这里要把原来 App.vue 里的 style 复制过来 */
.chat-container {
  max-width: 800px;
  margin: 0 auto;
  height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: #f5f7fa;
}
.header {
  padding: 20px;
  background: white;
  border-bottom: 1px solid #eee;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}
.message {
  display: flex;
  margin-bottom: 20px;
}
.message.user {
  flex-direction: row-reverse;
}
.avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: #e0e0e0;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 10px;
}
.content {
  background: white;
  padding: 10px 15px;
  border-radius: 8px;
  max-width: 70%;
  box-shadow: 0 2px 4px rgba(0,0,0,0.05);
}
.message.user .content {
  background: #165dff;
  color: white;
}
.input-area {
  padding: 20px;
  background: white;
  border-top: 1px solid #eee;
  display: flex;
  align-items: center;
}
.thinking-box {
  background: #f0f9ff;
  border-left: 3px solid #165dff;
  padding: 8px;
  margin-bottom: 8px;
  font-size: 0.9em;
  color: #666;
}
.think-title {
  font-weight: bold;
  margin-bottom: 4px;
}
</style>