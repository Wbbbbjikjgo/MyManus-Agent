<template>
    <div class="main-page">
        <div class="messages">
            <div v-for="(message, index) in messages" :key="index" :class="message.type">
                <strong>{{ message.type === 'user' ? 'User' : 'Server' }}:</strong>
                <span>{{ message.text }}
                    <a v-if="message.imageUrl" :href="message.imageUrl" target="_blank"><br /><img
                            :src="message.imageUrl" /></a>
                    <a v-if="message.fileUrl" :href="message.fileUrl" target="_blank"><br />👇download👇</a>
                </span>
            </div>
        </div>
        <div class="message-input">
            <input type="text" v-model="newMessage" placeholder="Type your message..." @keyup.enter="sendMessage" />
            <button @click="sendMessage" :disabled="disableInput">Send</button>
        </div>
    </div>
</template>

<script>
import { Client } from '@stomp/stompjs';

export default {
    data() {
        return {
            disableInput: false,
            isVisible: true,
            messages: [
                { type: 'user', text: 'Hello, server!' },
                { type: 'server', text: 'Hello, user!' },
            ],
            newMessage: '',
            stompClient: null
        };
    },
    methods: {
        sendMessage() {
            if (this.newMessage.trim()) {
                var msg = { type: 'user', text: this.newMessage };
                this.messages.push(msg);
                this.newMessage = '';
                this.stompClient.publish({ destination: '/app/enhanced-dialog', body: JSON.stringify(msg) });
                this.scrollToBottom();
            }
        },
        closeDialog() {
            this.isVisible = false;
        },
        handleMessage(playload) {
            const message = JSON.parse(playload.body);
            if (message.text) {
                this.messages.push(message);
                this.scrollToBottom();
            }
            if (message.meta) {
                if (message.meta.serverStatusHint == 0) {
                    this.disableInput = false;
                } else if (message.meta.serverStatusHint == 1) {
                    this.disableInput = true;
                }
            }
        },
        scrollToBottom() {
            this.$nextTick(() => {
                const messagesContainer = this.$el.querySelector('.messages');
                messagesContainer.scrollTop = messagesContainer.scrollHeight;
            });
        }
    },
    created() {
        console.log("Starting connection to WebSocket Server")
        this.stompClient = new Client({
            brokerURL: 'ws://localhost:18081/bs-dialog-websocket'
        });
        this.stompClient.onConnect = (frame) => {
            console.log('Connected: ' + frame);
            this.stompClient.subscribe('/user/queue/dialog', this.handleMessage);
        };
        this.stompClient.onWebSocketError = (error) => {
            console.error('Error with websocket', error);
        };

        this.stompClient.onStompError = (frame) => {
            console.error('Broker reported error: ' + frame.headers['message']);
            console.error('Additional details: ' + frame.body);
        };
        this.stompClient.activate();
    }
};
</script>

<style scoped>
img {
    max-width: 100px;
    max-height: 100px;
    margin-left: 10px;
}

.main-page {
    display: flex;
    flex-direction: column;
    height: 90vh;
    width: 100%;
    background: #f5f5f5;
}

.messages {
    flex: 1;
    overflow-y: auto;
    padding: 10px;
    background: white;
    border-bottom: 1px solid #ddd;
}

.messages .user {
    text-align: right;
    margin: 5px 0;
}

.messages .user span {
    display: inline-block;
    background: #007bff;
    color: white;
    padding: 10px;
    border-radius: 15px;
    max-width: 70%;
    word-wrap: break-word;
}

.messages .server {
    text-align: left;
    margin: 5px 0;
}

.messages .server span {
    display: inline-block;
    background: #e5e5ea;
    color: black;
    padding: 10px;
    border-radius: 15px;
    max-width: 70%;
    word-wrap: break-word;
}

.message-input {
    display: flex;
    padding: 10px;
    background: #fff;
    border-top: 1px solid #ddd;
}

.message-input input {
    flex: 1;
    padding: 10px;
    border: 1px solid #ddd;
    border-radius: 4px;
    margin-right: 10px;
}

.message-input button {
    padding: 10px 20px;
    background: #007bff;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
}

.message-input button:hover {
    background: #0056b3;
}

.dialog {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
}

.dialog-overlay {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.5);
}

.dialog-content {
    position: relative;
    background: white;
    padding: 20px;
    border-radius: 8px;
    width: 300px;
    max-width: 90%;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
}

.messages {
    max-height: 100%;
    overflow-y: auto;
    margin-bottom: 10px;
}

.messages .user {
    text-align: right;
    color: blue;
}

.messages .server {
    text-align: left;
    color: green;
}

.dialog-actions {
    display: flex;
    gap: 10px;
}

.dialog-actions input {
    flex: 1;
    padding: 5px;
}

.dialog-actions button {
    padding: 5px 10px;
}

.main-page {
    display: flex;
    flex-direction: column;
    height: 90vh;
    width: 100%;
    background: #f5f5f5;
}

.messages {
    flex: 1;
    overflow-y: auto;
    padding: 10px;
    background: white;
    border-bottom: 1px solid #ddd;
}

.messages .user {
    text-align: right;
    color: blue;
    margin: 5px 0;
}

.messages .server {
    text-align: left;
    color: green;
    margin: 5px 0;
}

.message-input {
    display: flex;
    padding: 10px;
    background: #fff;
    border-top: 1px solid #ddd;
}

.message-input input {
    flex: 1;
    padding: 10px;
    border: 1px solid #ddd;
    border-radius: 4px;
    margin-right: 10px;
}

.message-input button {
    padding: 10px 20px;
    background: #007bff;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
}

.message-input button:hover {
    background: #0056b3;
}

.message-input button:disabled,
.message-input button[disabled] {
    border: 1px solid #999999;
    background-color: #cccccc;
    color: #666666;
}
</style>
