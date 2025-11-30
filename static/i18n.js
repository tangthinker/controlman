const translations = {
    en: {
        "app_name": "ControlMan",
        "subtitle": "Service Management Interface",
        "login_title": "ControlMan - Login",
        "dashboard_title": "ControlMan - Dashboard",
        "info_title": "ControlMan - Service Info",
        "username": "Username",
        "password": "Password",
        "sign_in": "Sign In",
        "logout": "Logout",
        "services": "Services",
        "add_service": "Add Service",
        "name": "Name",
        "status": "Status",
        "pid": "PID",
        "command": "Command",
        "last_started": "Last Started",
        "actions": "Actions",
        "loading": "Loading...",
        "no_services": "No services found.",
        "failed_load": "Failed to load services.",
        "cancel": "Cancel",
        "add": "Add",
        "service_logs": "Service Logs",
        "refresh": "Refresh",
        "close": "Close",
        "back": "Back",
        "service_details": "Service Details",
        "real_time_monitor": "Real-time Monitor",
        "updating_every_second": "Updating every second.",
        "cpu_usage": "CPU Usage",
        "memory_usage": "Memory Usage",
        "log_file_path": "Log File Path",
        "confirm_start": "Are you sure you want to start service \"{name}\"?",
        "confirm_stop": "Are you sure you want to stop service \"{name}\"?",
        "confirm_restart": "Are you sure you want to restart service \"{name}\"?",
        "confirm_delete": "Are you sure you want to DELETE service \"{name}\"? This cannot be undone.",
        "unknown_error": "Unknown error",
        "network_error": "Network error",
        "invalid_credentials": "Invalid credentials",
        "connection_failed": "Connection failed",
        "login_failed": "Login failed",
        "service_added": "Service added successfully",
        "failed_add": "Failed to add service",
        "no_logs": "No logs available.",
        "failed_logs": "Failed to fetch logs.",
        "status_running": "Running",
        "status_stopped": "Stopped",
        "status_failed": "Failed",
        "status_starting": "Starting",
        "status_stopping": "Stopping",
        "status_restarting": "Restarting",
        "status_unknown": "Unknown",
        "cpu_history": "CPU History",
        "memory_history": "Memory History",
        "resource_monitor": "Resource Monitor",
        "reset_charts": "Reset Charts",
        "start": "Start",
        "stop": "Stop",
        "restart": "Restart",
        "delete": "Delete",
        "logs": "Logs",
        "service_deleted": "Service deleted successfully"
    },
    zh: {
        "app_name": "ControlMan",
        "subtitle": "服务管理界面",
        "login_title": "ControlMan - 登录",
        "dashboard_title": "ControlMan - 仪表盘",
        "info_title": "ControlMan - 服务详情",
        "username": "用户名",
        "password": "密码",
        "sign_in": "登录",
        "logout": "退出",
        "services": "服务列表",
        "add_service": "添加服务",
        "name": "名称",
        "status": "状态",
        "pid": "PID",
        "command": "命令",
        "last_started": "上次启动",
        "actions": "操作",
        "loading": "加载中...",
        "no_services": "未找到服务。",
        "failed_load": "加载服务失败。",
        "cancel": "取消",
        "add": "添加",
        "service_logs": "服务日志",
        "refresh": "刷新",
        "close": "关闭",
        "back": "返回",
        "service_details": "服务详情",
        "real_time_monitor": "实时监控",
        "updating_every_second": "每秒更新",
        "cpu_usage": "CPU 使用率",
        "memory_usage": "内存使用",
        "log_file_path": "日志文件路径",
        "confirm_start": "确定要启动服务 \"{name}\" 吗？",
        "confirm_stop": "确定要停止服务 \"{name}\" 吗？",
        "confirm_restart": "确定要重启服务 \"{name}\" 吗？",
        "confirm_delete": "确定要删除服务 \"{name}\" 吗？此操作不可撤销。",
        "unknown_error": "未知错误",
        "network_error": "网络错误",
        "invalid_credentials": "凭证无效",
        "connection_failed": "连接失败",
        "login_failed": "登录失败",
        "service_added": "服务添加成功",
        "failed_add": "添加服务失败",
        "no_logs": "暂无日志。",
        "failed_logs": "获取日志失败。",
        "status_running": "运行中",
        "status_stopped": "已停止",
        "status_failed": "失败",
        "status_starting": "启动中",
        "status_stopping": "停止中",
        "status_restarting": "重启中",
        "status_unknown": "未知",
        "cpu_history": "CPU 历史曲线",
        "memory_history": "内存历史曲线",
        "resource_monitor": "资源监控",
        "reset_charts": "重置图表",
        "start": "启动",
        "stop": "停止",
        "restart": "重启",
        "delete": "删除",
        "logs": "日志",
        "service_deleted": "服务删除成功"
    }
};

class I18n {
    constructor() {
        this.lang = localStorage.getItem('cm_lang') || 'zh'; // Default to Chinese as per request context "support bilingual" usually implies adding native lang
    }

    setLanguage(lang) {
        this.lang = lang;
        localStorage.setItem('cm_lang', lang);
        this.apply();
        // Trigger a custom event so other scripts can react (e.g. re-render tables)
        window.dispatchEvent(new CustomEvent('languageChanged', { detail: { lang } }));
    }

    t(key, params = {}) {
        let text = (translations[this.lang] && translations[this.lang][key]) || 
                   (translations['en'] && translations['en'][key]) || 
                   key;
        for (const [k, v] of Object.entries(params)) {
            text = text.replace(`{${k}}`, v);
        }
        return text;
    }

    apply() {
        // Update document title if corresponding key exists
        // We need to know which page we are on, or just try generic keys
        // Simple approach: check specific ID or rely on static translation
        
        document.querySelectorAll('[data-i18n]').forEach(el => {
            const key = el.getAttribute('data-i18n');
            if (key) {
                el.textContent = this.t(key);
            }
        });

        document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
            const key = el.getAttribute('data-i18n-placeholder');
            if (key) {
                el.placeholder = this.t(key);
            }
        });

        // Update language switcher UI
        const enBtn = document.getElementById('lang-en');
        const zhBtn = document.getElementById('lang-zh');
        if (enBtn && zhBtn) {
            const activeClass = ['text-indigo-600', 'font-bold'];
            const inactiveClass = ['text-gray-500'];
            
            if (this.lang === 'en') {
                enBtn.classList.add(...activeClass);
                enBtn.classList.remove(...inactiveClass);
                zhBtn.classList.remove(...activeClass);
                zhBtn.classList.add(...inactiveClass);
            } else {
                zhBtn.classList.add(...activeClass);
                zhBtn.classList.remove(...inactiveClass);
                enBtn.classList.remove(...activeClass);
                enBtn.classList.add(...inactiveClass);
            }
        }
    }
}

const i18n = new I18n();
window.i18n = i18n;

document.addEventListener('DOMContentLoaded', () => {
    i18n.apply();
});
