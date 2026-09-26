#!/usr/bin/env python3
"""验证已提交窗口输出，以及删除本次本地状态后的 changelog 恢复。"""
import os,pathlib,re,shutil,subprocess,uuid
ROOT=pathlib.Path(__file__).resolve().parent.parent
TOKEN='streams-test-'+uuid.uuid4().hex[:12];GROUP=TOKEN+'-stats'
ENV=dict(os.environ,KAFKA_TOPIC=TOKEN,KAFKA_GROUP=GROUP)
BROKERS=ENV.get('KAFKA_BROKERS','127.0.0.1:9092')
def cli(name):return shutil.which(name) or shutil.which(name+'.sh') or name
def demo(*args):
 p=subprocess.run(['./run.sh']+list(args),cwd=ROOT,env=ENV,capture_output=True,text=True,timeout=150)
 assert p.returncode==0,(args,p.stdout,p.stderr)
 return p.stdout
def totals(expected):
 p=subprocess.run([cli('kafka-console-consumer'),'--bootstrap-server',BROKERS,'--topic',TOKEN+'.paid-totals','--from-beginning','--timeout-ms','5000','--command-property','isolation.level=read_committed','--formatter-property','print.key=true'],capture_output=True,text=True,timeout=30)
 values={}
 for line in p.stdout.splitlines():
  match=re.fullmatch(r'(\d{4}-\S+)\t(\d+):(\d+)',line)
  if match:values[match[1]]=(int(match[2]),int(match[3]))
 assert sum(v[0] for v in values.values())==expected,p.stdout+'\n'+p.stderr
 assert sum(v[1] for v in values.values())==expected*1000,p.stdout
try:
 demo('init');demo('produce','--prefix','first')
 demo('streams','--timeout','40s');totals(3)
 # 只删除本脚本随机 application.id 的本地缓存，以验证远端 changelog 恢复。
 state=ROOT/'.cache/streams'/GROUP
 if state.exists():shutil.rmtree(state)
 demo('produce','--prefix','second')
 demo('streams','--timeout','40s');totals(6)
 print('PASS Streams committed output and changelog recovery')
finally:
 p=subprocess.run([cli('kafka-topics'),'--bootstrap-server',BROKERS,'--list'],capture_output=True,text=True,timeout=30)
 for topic in p.stdout.splitlines():
  if topic==TOKEN or topic==TOKEN+'.paid-totals' or topic.startswith(GROUP+'-'):
   subprocess.run([cli('kafka-topics'),'--bootstrap-server',BROKERS,'--delete','--topic',topic],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=30)
 state=ROOT/'.cache/streams'/GROUP
 if state.exists():shutil.rmtree(state)
