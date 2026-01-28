import asyncio
import json
import os
import socket
import sqlite3
import pandas as pd
import requests
import logging
import datetime
import time
from collections import deque
from contextlib import asynccontextmanager, contextmanager
from fastapi import FastAPI
from fastapi.responses import HTMLResponse, JSONResponse

# ================= 配置区 =================
API_USER = "admin" 
API_PASS = "nyzz001" 
DATA_FILE = "bot_sniper_data.json" # 🔥 旧数据文件 (用于迁移)
DATABASE_FILE = "bot_sniper.db" # 🔥 SQLite 数据库文件
# ========================================

# 进出场规则
ENTRY_CONDITION = 2  # 虚盘连赢 2 把 -> 进实盘
EXIT_CONDITION = 1   # 实盘连输 1 把 -> 退回虚盘

BET_LABELS = ['红奔驰','绿奔驰','黄奔驰','红宝马','绿宝马','黄宝马','红奥迪','绿奥迪','黄奥迪','红大众','绿大众','黄大众']
REAL_ODDS = {'红奔驰':45,'绿奔驰':38,'黄奔驰':27,'红宝马':22,'绿宝马':16,'黄宝马':13,'红奥迪':12,'绿奥迪':10,'黄奥迪':6,'红大众':7,'绿大众':5,'黄大众':4}
SMALL_CARS = ['红大众', '绿大众', '黄大众', '红奥迪', '绿奥迪', '黄奥迪']
BIG_CARS = ['红奔驰', '绿奔驰', '黄奔驰', '红宝马', '绿宝马', '黄宝马']
SPECIAL_REWARDS = ['大三元', '大四喜', '极速狂飙', 'U型过弯', '全民送灯']

COLORS = {'红': [], '绿': [], '黄': []}
LOGOS = {'奔驰': [], '宝马': [], '奥迪': [], '大众': []}
for car in BET_LABELS:
    for c in COLORS: 
        if c in car: COLORS[c].append(car)
    for l in LOGOS:
        if l in car: LOGOS[l].append(car)

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# === 数据库管理器 ===
class DatabaseManager:
    def __init__(self, db_file=DATABASE_FILE):
        self.db_file = db_file
        self.init_db()
    
    @contextmanager
    def get_connection(self):
        conn = sqlite3.connect(self.db_file)
        conn.row_factory = sqlite3.Row
        try:
            yield conn
            conn.commit()
        except Exception as e:
            conn.rollback()
            logger.error(f"数据库错误: {e}")
            raise
        finally:
            conn.close()
    
    def init_db(self):
        """初始化数据库表结构"""
        with self.get_connection() as conn:
            cursor = conn.cursor()
            
            # 创建策略状态表
            cursor.execute('''
                CREATE TABLE IF NOT EXISTS strategies (
                    name TEXT PRIMARY KEY,
                    profit INTEGER DEFAULT 0,
                    real_profit INTEGER DEFAULT 0,
                    wins INTEGER DEFAULT 0,
                    count INTEGER DEFAULT 0,
                    state INTEGER DEFAULT 0,
                    v_streak INTEGER DEFAULT 0,
                    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                )
            ''')
            
            # 创建游戏历史表
            cursor.execute('''
                CREATE TABLE IF NOT EXISTS game_logs (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    round_id TEXT UNIQUE NOT NULL,
                    time TEXT NOT NULL,
                    result_name TEXT NOT NULL,
                    winners_json TEXT,
                    matrix TEXT NOT NULL,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                )
            ''')
            
            # 创建索引以加快查询
            cursor.execute('''
                CREATE INDEX IF NOT EXISTS idx_round_id ON game_logs(round_id)
            ''')
            cursor.execute('''
                CREATE INDEX IF NOT EXISTS idx_created_at ON game_logs(created_at DESC)
            ''')
            
            logger.info("✅ 数据库初始化完成")
    
    def save_strategy(self, name, data):
        """保存或更新策略状态"""
        with self.get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute('''
                INSERT INTO strategies (name, profit, real_profit, wins, count, state, v_streak, updated_at)
                VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
                ON CONFLICT(name) DO UPDATE SET
                    profit = excluded.profit,
                    real_profit = excluded.real_profit,
                    wins = excluded.wins,
                    count = excluded.count,
                    state = excluded.state,
                    v_streak = excluded.v_streak,
                    updated_at = CURRENT_TIMESTAMP
            ''', (name, data['profit'], data['real_profit'], data['wins'], 
                  data['count'], data['state'], data['v_streak']))
    
    def load_strategies(self):
        """加载所有策略状态"""
        with self.get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute('SELECT * FROM strategies')
            rows = cursor.fetchall()
            
            result = {}
            for row in rows:
                result[row['name']] = {
                    'profit': row['profit'],
                    'real_profit': row['real_profit'],
                    'wins': row['wins'],
                    'count': row['count'],
                    'state': row['state'],
                    'v_streak': row['v_streak']
                }
            return result
    
    def save_game_log(self, log):
        """保存游戏历史记录"""
        with self.get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute('''
                INSERT OR IGNORE INTO game_logs (round_id, time, result_name, winners_json, matrix)
                VALUES (?, ?, ?, ?, ?)
            ''', (log['id'], log['time'], log['res'], 
                  log.get('winners_json', ''), json.dumps(log['matrix'], ensure_ascii=False)))
    
    def get_logs(self, limit=50, offset=0):
        """分页获取历史记录"""
        with self.get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute('''
                SELECT round_id as id, time, result_name as res, winners_json, matrix
                FROM game_logs
                ORDER BY id DESC
                LIMIT ? OFFSET ?
            ''', (limit, offset))
            rows = cursor.fetchall()
            
            logs = []
            for row in rows:
                logs.append({
                    'id': row['id'],
                    'time': row['time'],
                    'res': row['res'],
                    'winners_json': row['winners_json'] or '',
                    'matrix': json.loads(row['matrix']) if row['matrix'] else {}
                })
            return logs
    
    def get_total_logs_count(self):
        """获取总记录数"""
        with self.get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute('SELECT COUNT(*) as count FROM game_logs')
            return cursor.fetchone()['count']
    
    def migrate_from_json(self, json_file):
        """从 JSON 文件迁移数据到 SQLite"""
        if not os.path.exists(json_file):
            logger.info("未找到旧数据文件，跳过迁移")
            return
        
        try:
            with open(json_file, 'r', encoding='utf-8') as f:
                data = json.load(f)
            
            # 迁移策略数据
            strategies = data.get('strategies', {})
            for name, strat_data in strategies.items():
                # 确保包含所有必需字段
                save_data = {
                    'profit': strat_data.get('profit', 0),
                    'real_profit': strat_data.get('real_profit', 0),
                    'wins': strat_data.get('wins', 0),
                    'count': strat_data.get('count', 0),
                    'state': 0,  # 迁移时重置状态
                    'v_streak': 0  # 迁移时重置连赢
                }
                self.save_strategy(name, save_data)
            
            # 迁移历史记录
            logs = data.get('logs', [])
            logger.info(f"开始迁移 {len(logs)} 条历史记录...")
            
            with self.get_connection() as conn:
                cursor = conn.cursor()
                for log in logs:
                    try:
                        cursor.execute('''
                            INSERT OR IGNORE INTO game_logs (round_id, time, result_name, winners_json, matrix)
                            VALUES (?, ?, ?, ?, ?)
                        ''', (log['id'], log['time'], log['res'], 
                              '', json.dumps(log.get('matrix', {}), ensure_ascii=False)))
                    except Exception as e:
                        logger.warning(f"迁移记录 {log.get('id')} 失败: {e}")
                        continue
            
            logger.info(f"✅ 数据迁移完成：{len(strategies)} 个策略，{len(logs)} 条记录")
            
            # 备份旧文件
            backup_file = json_file + '.backup'
            os.rename(json_file, backup_file)
            logger.info(f"✅ 旧数据文件已备份至: {backup_file}")
            
        except Exception as e:
            logger.error(f"数据迁移失败: {e}")

# === 核心工具 ===
def clean_name(n):
    if not isinstance(n, str): return "未知"
    n = n.strip()
    
    # 精确匹配特殊奖励
    for spec in SPECIAL_REWARDS:
        if spec in n: return n
    
    # 模糊匹配特殊奖励关键词（容错处理）
    special_keywords = {
        '三元': '大三元',
        '四喜': '大四喜', 
        '狂飙': '极速狂飙',
        '过弯': 'U型过弯',
        '送灯': '全民送灯'
    }
    for keyword, full_name in special_keywords.items():
        if keyword in n:
            return full_name
    
    # 匹配车型
    if n in BET_LABELS: return n
    for l in BET_LABELS:
        if len(l)==3 and l[0] in n and l[-2:] in n: return l
    return n

def parse_wins(winners_json, result_name):
    w = set()
    try:
        data = json.loads(winners_json) if isinstance(winners_json, str) and winners_json.startswith('[') else winners_json
        if isinstance(data, list):
            for item in data:
                raw_name = item.get('name', '') if isinstance(item, dict) else str(item)
                clean = clean_name(raw_name)
                if clean in BET_LABELS: w.add(clean)
                for spec in SPECIAL_REWARDS:
                    if spec in clean: w.add(clean)
    except: pass
    if not w:
        mn = clean_name(result_name)
        w.add(mn)
    return list(w)

def get_full_result_display(winners_json, result_name):
    wins = parse_wins(winners_json, result_name)
    specs = [x for x in wins if any(s in x for s in SPECIAL_REWARDS)]
    cars = [x for x in wins if x in BET_LABELS]
    if specs:
        main_spec = specs[0]
        if cars: return f"{main_spec} [{', '.join(cars)}]"
        return main_spec
    if cars: return ", ".join(cars)
    return clean_name(result_name)

def calc_profit(preds, act_name, winners_json):
    if not preds: return 0, False
    cost = 100 * len(preds)
    rev = 0; winning_items = parse_wins(winners_json, act_name); win_bool = False
    for p in preds:
        hit = False
        if p in winning_items: hit = True
        else:
            for winner in winning_items:
                if p in winner: hit = True; break
        if hit:
            win_bool = True
            rev += 100 * REAL_ODDS.get(p, 2)
    return rev - cost, win_bool

# === 策略引擎 ===
class StrategyEngine:
    def get_heat_scores(self, df, limit=30):
        scores = {l: 0.0 for l in BET_LABELS}
        recent = df.tail(limit)
        total = len(recent)
        for idx, row in enumerate(recent.iterrows()):
            wins = parse_wins(row[1].get('winners_json'), row[1]['result_name'])
            weight = 0.5 + (idx / total)
            for w in wins:
                for label in BET_LABELS:
                    if label in w: scores[label] += 1.0 * weight
        return scores

    def strat_hot_3(self, df):
        scores = self.get_heat_scores(df, 30)
        return sorted(BET_LABELS, key=lambda x: scores[x], reverse=True)[:3]
    
    def strat_balanced_4(self, df):
        scores = self.get_heat_scores(df, 30)
        big = sorted(BIG_CARS, key=lambda x: scores[x], reverse=True)[:1]
        small = sorted(SMALL_CARS, key=lambda x: scores[x], reverse=True)[:3]
        return big + small

class BotSystem:
    def __init__(self):
        self.u = API_USER; self.p = API_PASS
        self.running = True
        self.token = None; self.lid = None
        self.engine = StrategyEngine()
        self.last_update_time = time.time()
        self.last_result = ''  # 上期结果
        
        # 初始化数据库
        self.db = DatabaseManager()
        
        # 迁移旧数据（如果存在）
        self.db.migrate_from_json(DATA_FILE)
        
        # 初始化策略状态机
        # state: 0=观望(虚盘), 1=实盘
        # v_streak: 虚盘连赢次数
        # real_profit: 实盘累计盈利
        base_struct = {'pred': [], 'profit': 0, 'wins': 0, 'count': 0, 'cost': 0, 'state': 0, 'v_streak': 0, 'real_profit': 0}
        
        self.strategies = {
            '🔥 热门(3码)': {**base_struct, 'func': self.engine.strat_hot_3, 'cost': 300},
            '⚖️ 均衡(4码)': {**base_struct, 'func': self.engine.strat_balanced_4, 'cost': 400},
        }
        self.load_data()

    def load_data(self):
        """从数据库加载策略状态"""
        try:
            saved = self.db.load_strategies()
            for k, v in saved.items():
                if k in self.strategies:
                    self.strategies[k]['profit'] = v.get('profit', 0)
                    self.strategies[k]['real_profit'] = v.get('real_profit', 0)
                    self.strategies[k]['wins'] = v.get('wins', 0)
                    self.strategies[k]['count'] = v.get('count', 0)
                    self.strategies[k]['state'] = v.get('state', 0)
                    self.strategies[k]['v_streak'] = v.get('v_streak', 0)
            logger.info("✅ 策略状态加载完成")
        except Exception as e:
            logger.error(f"加载策略状态失败: {e}")

    def save_data(self):
        """保存策略状态到数据库"""
        try:
            for k, v in self.strategies.items():
                save_dict = {
                    'profit': v['profit'], 
                    'real_profit': v['real_profit'], 
                    'wins': v['wins'], 
                    'count': v['count'],
                    'state': v['state'],
                    'v_streak': v['v_streak']
                }
                self.db.save_strategy(k, save_dict)
        except Exception as e:
            logger.error(f"保存策略状态失败: {e}")

    async def login(self):
        try:
            r = requests.post('http://43.136.31.62:4173/api/login', json={'username':self.u,'password':self.p}, timeout=5)
            if r.status_code in [200, 201]:
                d = r.json().get('data')
                if isinstance(d, dict): self.token = d.get('accessToken') or d.get('token')
                elif isinstance(d, str): self.token = d
                if self.token: logger.info("✅ 登录成功"); return True
        except: pass
        return False

    async def fetch_data(self):
        if not self.token: await self.login()
        if not self.token: return []
        headers = {'Authorization': f'Bearer {self.token}'}
        try:
            r = requests.get('http://43.136.31.62:4173/api/ymbz/records', headers=headers, params={'limit':50,'page':1}, timeout=5)
            if r.status_code == 200: return r.json().get('data', {}).get('items', [])
            if r.status_code == 401: await self.login()
        except: pass
        return []

    async def loop(self):
        logger.info(f"🚀 V52.0 狙击手版启动 (虚实切换)")
        while self.running:
            try:
                items = await self.fetch_data()
                if items:
                    df = pd.DataFrame(items)
                    df['round_id'] = pd.to_numeric(df['round_id'])
                    df = df.sort_values('round_id')
                    latest = df.iloc[-1]
                    lid = str(latest['round_id'])
                    
                    if self.lid != lid:
                        self.last_update_time = time.time()
                        now_time = datetime.datetime.now().strftime("%H:%M:%S")
                        rn = clean_name(latest['result_name'])
                        
                        # 🔍 诊断日志：检查外部API返回的原始数据
                        logger.info(f"🔍 原始数据 - result_name: {latest['result_name']}, winners_json: {latest.get('winners_json', 'MISSING')}")
                        
                        full_res = get_full_result_display(latest.get('winners_json'), latest['result_name'])
                        
                        # 🔍 诊断日志：检查处理后的结果
                        logger.info(f"🔍 处理后 - full_res: {full_res}")
                        
                        matrix_snapshot = {}
                        
                        # === 1. 结算阶段 ===
                        for name, strat in self.strategies.items():
                            p_prof = 0; is_win = False
                            current_pred = list(strat['pred']) if strat['pred'] else []
                            
                            # 计算理论盈亏 (无论是否实盘)
                            if current_pred:
                                p_prof, is_win = calc_profit(current_pred, rn, latest.get('winners_json'))
                                strat['profit'] += p_prof # 理论总账
                                strat['count'] += 1
                                if is_win: strat['wins'] += 1
                            
                            # 🔥 核心逻辑：虚实切换 🔥
                            current_state = strat['state']
                            
                            # 记录快照 (用于前端展示)
                            matrix_snapshot[name] = {
                                'pred': current_pred,
                                'profit': int(p_prof),
                                'state': current_state, # 0=观, 1=实
                                'real_change': 0
                            }

                            if current_state == 1: # 实盘中
                                strat['real_profit'] += p_prof # 记入实盘账本
                                matrix_snapshot[name]['real_change'] = int(p_prof)
                                
                                if p_prof > 0: # 赢了
                                    # 乘胜追击，保持实盘
                                    pass 
                                else: # 输了
                                    # 🚨 立即止损，退回虚盘
                                    strat['state'] = 0
                                    strat['v_streak'] = 0 # 重置连赢计数
                            
                            else: # 观望中
                                if p_prof > 0:
                                    strat['v_streak'] += 1
                                else:
                                    strat['v_streak'] = 0
                                
                                # 🚨 触发进场：虚盘连赢达标
                                if strat['v_streak'] >= ENTRY_CONDITION:
                                    strat['state'] = 1

                        logger.info(f"💰 结算 {lid}")
                        log_entry = {'time': now_time, 'id': lid, 'res': full_res, 'matrix': matrix_snapshot, 'winners_json': latest.get('winners_json', '')}
                        
                        # 保存到数据库
                        self.db.save_game_log(log_entry)
                        self.save_data()
                        self.last_result = full_res

                        # === 2. 预测阶段 ===
                        self.lid = lid
                        for name, strat in self.strategies.items():
                            strat['pred'] = strat['func'](df)
                        
                        # === 3. 数据已保存到数据库，前端通过 HTTP API 获取 ===
            except Exception as e: logger.error(f"Loop: {e}")
            await asyncio.sleep(2)

bot = BotSystem()

@asynccontextmanager
async def lifespan(app: FastAPI):
    task = asyncio.create_task(bot.loop())
    yield
    task.cancel()

app = FastAPI(lifespan=lifespan)

@app.get("/api/status")
async def get_status():
    """获取当前状态和排行榜"""
    try:
        t_pass = int(time.time() - bot.last_update_time)
        countdown = max(0, 34 - t_pass)
        
        leaderboard = []
        for name, strat in bot.strategies.items():
            rate = int((strat['wins'] / strat['count'] * 100)) if strat['count'] > 0 else 0
            leaderboard.append({
                'name': name, 
                'profit': int(strat['real_profit']), 
                'total_profit': int(strat['profit']),
                'rate': rate, 
                'state': strat['state'], 
                'next': strat['pred']
            })
        leaderboard.sort(key=lambda x: x['profit'], reverse=True)
        
        # 获取最新的历史记录用于首页显示
        logs = bot.db.get_logs(limit=50, offset=0)
        
        return JSONResponse({
            'lid': str(bot.lid or ""),
            'next_lid': str(int(bot.lid) + 1) if bot.lid else "",
            'last_res': bot.last_result,
            'time_passed': t_pass,
            'countdown': countdown,
            'leaderboard': leaderboard,
            'logs': logs
        })
    except Exception as e:
        logger.error(f"API错误: {e}")
        return JSONResponse({'error': str(e)}, status_code=500)

@app.get("/api/logs")
async def get_logs(page: int = 1, size: int = 50):
    """分页获取历史记录"""
    try:
        if page < 1: page = 1
        if size < 1 or size > 200: size = 50
        
        offset = (page - 1) * size
        logs = bot.db.get_logs(limit=size, offset=offset)
        total = bot.db.get_total_logs_count()
        
        return JSONResponse({
            'total': total,
            'page': page,
            'size': size,
            'total_pages': (total + size - 1) // size,
            'logs': logs
        })
    except Exception as e:
        logger.error(f"API错误: {e}")
        return JSONResponse({'error': str(e)}, status_code=500)

@app.get("/api/predictions")
async def get_predictions():
    """获取下一期预测（仅实盘策略）"""
    try:
        # 使用 set 去重所有实盘策略的预测项
        all_items = set()
        
        # 遍历所有策略，筛选实盘状态的策略
        for name, strat in bot.strategies.items():
            if strat['state'] == 1:  # 只返回实盘状态的策略
                pred_items = strat['pred'] if strat['pred'] else []
                all_items.update(pred_items)
        
        # 转换为 map 格式，每项金额固定为 100
        predictions = {item: 100 for item in all_items}
        
        # 计算下注期号
        next_round = str(int(bot.lid) + 1) if bot.lid else ""
        
        return JSONResponse({
            'round': next_round,
            'predictions': predictions
        })
    except Exception as e:
        logger.error(f"API错误: {e}")
        return JSONResponse({'error': str(e)}, status_code=500)

@app.get("/")
async def get(): return HTMLResponse(open("index.html", "r", encoding='utf-8').read())

if __name__ == "__main__":
    import uvicorn
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    try: s.connect(('8.8.8.8', 80)); ip = s.getsockname()[0]
    except: ip = '127.0.0.1'
    finally: s.close()
    print(f"📱 狙击手地址: http://{ip}:8001")
    uvicorn.run(app, host="0.0.0.0", port=8001)