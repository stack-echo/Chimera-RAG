<script setup>
import { ref, reactive, nextTick } from 'vue';
import { fetchEventSource } from '@microsoft/fetch-event-source';
import MarkdownIt from 'markdown-it';
import { IconSend, IconUpload } from '@arco-design/web-vue/es/icon';

// --- 工具初始化 ---
const md = new MarkdownIt();

// --- 状态定义 ---
const inputValue = ref('');
const loading = ref(false);
const chatContainer = ref(null);
const fileList = ref([]); // 上传文件列表

// 消息列表 (默认一条欢迎语)
const messages = reactive([
  {
    role: 'assistant',
    content: '你好！我是 Chimera EHS 智能助手。请上传文档或直接提问。',
    html: md.render('你好！我是 Chimera EHS 智能助手。请上传文档或直接提问。')
  }
]);

// --- 核心逻辑 1: 发送消息 (SSE 流式) ---
const sendMessage = async () => {
  if (!inputValue.value.trim() || loading.value) return;

  const userQuery = inputValue.value;
  // 1. 添加用户消息
  messages.push({ role: 'user', content: userQuery, html: userQuery });
  inputValue.value = '';
  loading.value = true;

  // 2. 添加一个空的 AI 消息占位
  const assistantMsgIndex = messages.push({ role: 'assistant', content: '', html: '' }) - 1;

  // 3. 发起 SSE 请求
  try {
    await fetchEventSource('/api/v1/chat/stream', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query: userQuery }),

      onmessage(msg) {
        // 如果后端发来的是 error 事件
        if (msg.event === 'error') {
          console.error('Error:', msg.data);
          return;
        }

        // 处理正常消息
        // 后端格式： "THINKing: xxx" 或 "ANSWER: xxx"
        // 这里做一个简单的处理，把前缀去掉，直接拼接
        let text = msg.data;

        // 简单清洗一下前缀 (你可以根据后端实际返回调整)
        if (text.startsWith('THINKing:')) text = `> *${text.substring(9)}*\n\n`;
        if (text.startsWith('ANSWER:')) text = text.substring(7);

        // 实时追加内容
        messages[assistantMsgIndex].content += text;
        // 实时渲染 Markdown
        messages[assistantMsgIndex].html = md.render(messages[assistantMsgIndex].content);

        scrollToBottom();
      },
      onclose() {
        loading.value = false;
      },
      onerror(err) {
        console.log('SSE Error:', err);
        loading.value = false;
        throw err; // 抛出错误以停止重试
      }
    });
  } catch (err) {
    messages[assistantMsgIndex].html += `<br/><span style="color:red">请求出错: ${err.message}</span>`;
    loading.value = false;
  }
};

// 滚动到底部
const scrollToBottom = () => {
  nextTick(() => {
    if (chatContainer.value) {
      chatContainer.value.scrollTop = chatContainer.value.scrollHeight;
    }
  });
};

// --- 核心逻辑 2: 文件上传 ---
// 注意：action 直接填后端地址，或者通过 Vite 代理
const uploadAction = '/api/v1/upload';

const onUploadSuccess = (fileItem) => {
  messages.push({
    role: 'system',
    content: `文件 ${fileItem.name} 上传成功！`,
    html: `✅ *文件 ${fileItem.name} 已加入知识库，正在解析中...*`
  });
};
</script>

<template>
  <a-layout class="layout-container">
    <a-layout-sider theme="dark" :width="260">
      <div class="logo">🦄 Chimera RAG</div>

      <div class="upload-area">
        <a-upload
          draggable
          :action="uploadAction"
          :file-list="fileList"
          @success="onUploadSuccess"
          name="file"
        />
        <p class="tip">支持 PDF 文档上传</p>
      </div>

      <a-menu theme="dark" :default-selected-keys="['1']">
        <a-menu-item key="1">🤖 智能问答</a-menu-item>
        <a-menu-item key="2">📚 知识库管理</a-menu-item>
      </a-menu>
    </a-layout-sider>

    <a-layout>
      <a-layout-header class="header">
        EHS 安全合规助手 (DeepSeek V3 Powered)
      </a-layout-header>

      <a-layout-content class="chat-wrapper">
        <div class="message-list" ref="chatContainer">
          <div
            v-for="(msg, index) in messages"
            :key="index"
            :class="['message-item', msg.role]"
          >
            <div class="avatar">{{ msg.role === 'user' ? '👨‍💻' : '🦄' }}</div>
            <div class="bubble" v-html="msg.html"></div>
          </div>
        </div>

        <div class="input-area">
          <a-textarea
            v-model="inputValue"
            placeholder="请输入您的问题... (Enter 发送)"
            :auto-size="{ minRows: 2, maxRows: 5 }"
            @keydown.enter.prevent="sendMessage"
          />
          <a-button type="primary" class="send-btn" @click="sendMessage" :loading="loading">
            <template #icon><icon-send /></template>
            发送
          </a-button>
        </div>
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<style scoped>
.layout-container { height: 100vh; }
.logo { height: 60px; line-height: 60px; text-align: center; color: #fff; font-size: 18px; font-weight: bold; border-bottom: 1px solid #333; }
.upload-area { padding: 20px; text-align: center; border-bottom: 1px solid #333; }
.tip { color: #888; font-size: 12px; margin-top: 10px; }
.header { background: #fff; border-bottom: 1px solid #eee; padding: 0 20px; font-weight: bold; display: flex; align-items: center; }

/* 聊天区域 */
.chat-wrapper { display: flex; flex-direction: column; background: #f5f7fa; }
.message-list { flex: 1; overflow-y: auto; padding: 20px; }
.message-item { display: flex; margin-bottom: 20px; }
.avatar { width: 40px; height: 40px; background: #ddd; border-radius: 50%; text-align: center; line-height: 40px; margin-right: 10px; flex-shrink: 0; }
.message-item.user { flex-direction: row-reverse; }
.message-item.user .avatar { margin-right: 0; margin-left: 10px; background: #165dff; color: #fff; }
.bubble { background: #fff; padding: 10px 15px; border-radius: 8px; max-width: 70%; line-height: 1.6; box-shadow: 0 1px 2px rgba(0,0,0,0.1); }
.message-item.user .bubble { background: #e8f3ff; }

/* 输入框 */
.input-area { background: #fff; padding: 20px; border-top: 1px solid #eee; display: flex; gap: 10px; align-items: flex-end; }
.send-btn { height: auto; padding: 10px 20px; }
</style>