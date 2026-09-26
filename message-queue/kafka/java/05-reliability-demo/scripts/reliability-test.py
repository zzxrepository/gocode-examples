#!/usr/bin/env python3
"""先执行 schema.sql、设置 MySQL 连接；只清理本次随机 ID 的课程数据。"""
import os,pathlib,shlex,shutil,subprocess,uuid
ROOT=pathlib.Path(__file__).resolve().parent.parent
TOKEN='verify-'+uuid.uuid4().hex[:12]
ENV=dict(os.environ,KAFKA_TOPIC=TOKEN,KAFKA_GROUP=TOKEN)
BROKERS=ENV.get('KAFKA_BROKERS','127.0.0.1:9092')
MYSQL=['mysql']+shlex.split(os.environ.get('MYSQL_TEST_ARGS',''))+['--batch','--skip-column-names','kafka_order_demo']
CREATED=False

def sql(query):
 return subprocess.check_output(MYSQL+['-e',query],text=True).strip()
def run(*args,fail=False,extra=None):
 p=subprocess.run(['./run.sh']+list(args),cwd=ROOT,env=dict(ENV,**(extra or {})),capture_output=True,text=True,timeout=150)
 assert (p.returncode!=0)==fail,(args,p.stdout,p.stderr)
 return p.stdout+p.stderr
try:
 sql('SELECT 1')
 run('init');CREATED=True
 run('produce','--duplicate','--prefix',TOKEN)
 s=run('project','--max','12',fail=True,extra={'FAIL_AFTER_DB':'1'})
 assert 'injected failure after DB commit' in s,s
 s=run('project','--max','12');assert 'DUPLICATE' in s,s
 assert sql(f"SELECT COUNT(*) FROM processed_events WHERE consumer_name='{TOKEN}'")=='6'
 assert sql(f"SELECT COUNT(*) FROM order_projection WHERE consumer_name='{TOKEN}' AND status='OrderPaid' AND version=2")=='3'
 # 相同 Topic/Group 接着处理 Outbox 的七条记录。
 run('place','--prefix',TOKEN+'-outbox')
 s=run('relay',fail=True,extra={'FAIL_AFTER_SEND':'1'});assert 'injected failure after Kafka ACK' in s,s
 run('relay')
 s=run('project','--max','7');assert 'DUPLICATE' in s,s
 assert sql(f"SELECT COUNT(*) FROM order_outbox WHERE topic='{TOKEN}' AND published=1")=='6'
 assert sql(f"SELECT COUNT(*) FROM processed_events WHERE consumer_name='{TOKEN}'")=='12'
 assert sql(f"SELECT COUNT(*) FROM order_projection WHERE consumer_name='{TOKEN}' AND status='OrderPaid' AND version=2")=='6'
 print('PASS database crash replay, deduplication, Outbox resend:',ROOT.parent.name)
finally:
 sql(f"DELETE FROM processed_events WHERE consumer_name='{TOKEN}'; DELETE FROM order_projection WHERE consumer_name='{TOKEN}'; DELETE FROM order_outbox WHERE topic='{TOKEN}'; DELETE FROM source_orders WHERE order_id LIKE '{TOKEN}-outbox-%'")
 if CREATED:
  cli=shutil.which('kafka-topics') or shutil.which('kafka-topics.sh')
  if cli:subprocess.run([cli,'--bootstrap-server',BROKERS,'--delete','--topic',TOKEN],check=True,stdout=subprocess.DEVNULL)
