// Admin log management functionality
function adminLogManagement() {
    return {
        logs: [],
        total: 0,
        currentPage: 1,
        pageSize: 10,
        searchForm: {
            username: '',
            action: '',
            resource: '',
            startDate: '',
            endDate: ''
        },
        loading: false,
        init() {
            this.fetchLogs();
        },
        async fetchLogs() {
            try {
                const params = new URLSearchParams({
                    page: this.currentPage,
                    pageSize: this.pageSize,
                    ...this.searchForm
                });

                const response = await fetch(`/admin/api/logs?${params}`);
                if (!response.ok) throw new Error('获取日志列表失败');
                const data = await response.json();
                this.logs = data.data;
                this.total = data.total;
            } catch (error) {
                Alpine.store('notification').show(error.message, 'error');
            }
        },
        search() {
            this.currentPage = 1;
            this.fetchLogs();
        },
        async clearLogs() {
            if (!confirm('确定要清理日志吗？此操作将删除所选日期之前的所有日志。')) return;

            try {
                const response = await fetch(`/admin/api/logs?beforeDate=${this.searchForm.endDate || new Date().toISOString()}`, {
                    method: 'DELETE'
                });

                if (!response.ok) {
                    const error = await response.json();
                    throw new Error(error.error || '清理日志失败');
                }

                Alpine.store('notification').show('日志清理成功', 'success');
                this.fetchLogs();
            } catch (error) {
                Alpine.store('notification').show(error.message, 'error');
            }
        },
        previousPage() {
            if (this.currentPage > 1) {
                this.currentPage--;
                this.fetchLogs();
            }
        },
        nextPage() {
            if (this.currentPage * this.pageSize < this.total) {
                this.currentPage++;
                this.fetchLogs();
            }
        },
        getActionText(action) {
            const actionMap = {
                'create': '创建',
                'update': '更新',
                'delete': '删除'
            };
            return actionMap[action] || action;
        },
        getResourceText(resource) {
            const resourceMap = {
                'user': '用户',
                'role': '角色',
                'permission': '权限'
            };
            return resourceMap[resource] || resource;
        },
        formatDate(date) {
            if (!date) return '';
            return new Date(date).toLocaleString('zh-CN', {
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit',
                hour12: false
            });
        }
    }
} 