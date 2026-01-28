/**
 * API接口封装模块
 * 负责所有与后端的通信
 */

const API = {
  // API基础URL（相对路径，自动使用当前域名）
  baseURL: '',
  
  /**
   * 获取实时状态和排行榜
   * @returns {Promise<Object>} 包含lid, next_lid, last_res, countdown, leaderboard, logs
   */
  async getStatus() {
    try {
      const response = await fetch(`${this.baseURL}/api/status`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      return {
        success: true,
        data
      };
    } catch (error) {
      console.error('获取状态失败:', error);
      return {
        success: false,
        error: error.message,
        data: null
      };
    }
  },
  
  /**
   * 获取历史记录（分页）
   * @param {number} page - 页码（从1开始）
   * @param {number} size - 每页数量
   * @returns {Promise<Object>} 包含total, page, size, total_pages, logs
   */
  async getLogs(page = 1, size = 50) {
    try {
      const response = await fetch(`${this.baseURL}/api/logs?page=${page}&size=${size}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      return {
        success: true,
        data
      };
    } catch (error) {
      console.error('获取历史记录失败:', error);
      return {
        success: false,
        error: error.message,
        data: null
      };
    }
  },
  
  /**
   * 获取下期预测（仅实盘策略）
   * @returns {Promise<Object>} 包含round, predictions
   */
  async getPredictions() {
    try {
      const response = await fetch(`${this.baseURL}/api/predictions`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      return {
        success: true,
        data
      };
    } catch (error) {
      console.error('获取预测失败:', error);
      return {
        success: false,
        error: error.message,
        data: null
      };
    }
  },
  
  /**
   * 轮询获取状态（带节流）
   * @param {Function} callback - 数据更新回调函数
   * @param {number} interval - 轮询间隔（毫秒），默认2000ms
   * @returns {Function} 停止轮询的函数
   */
  startPolling(callback, interval = 2000) {
    let timerId = null;
    let isPolling = false;
    
    const poll = async () => {
      if (isPolling) return; // 防止重复请求
      
      isPolling = true;
      const result = await this.getStatus();
      
      if (result.success) {
        callback(result.data);
      }
      
      isPolling = false;
      timerId = setTimeout(poll, interval);
    };
    
    // 立即执行一次
    poll();
    
    // 返回停止轮询的函数
    return () => {
      if (timerId) {
        clearTimeout(timerId);
        timerId = null;
      }
    };
  }
};

// 导出API对象
if (typeof module !== 'undefined' && module.exports) {
  module.exports = API;
}
