#!/usr/bin/env python3
"""在独立临时 Topic 上验证本章；只删除本脚本成功创建的 Topic。"""
import json,os,pathlib,re,shutil,subprocess,sys,time,uuid
ROOT=pathlib.Path(__file__).resolve().parent.parent
LANG='go' if (ROOT/'go.mod').exists() else 'java'
CHAPTER=int(ROOT.name[:2]);TOKEN='course-test-'+uuid.uuid4().hex[:12]
TOPIC=TOKEN;BROKERS=os.environ.get('KAFKA_BROKERS','127.0.0.1:9092')
ENV=dict(os.environ,KAFKA_TOPIC=TOPIC,KAFKA_GROUP=TOKEN,KAFKA_BROKERS=BROKERS)
CREATED=[]
def run(args,ok=True,extra=None):
 p=subprocess.run(args,cwd=ROOT,env=dict(ENV,**(extra or {})),text=True,capture_output=True,timeout=180)
 if (p.returncode==0)!=ok:raise AssertionError(f'{args}: exit={p.returncode}\n{p.stdout}\n{p.stderr}')
 return p.stdout

def main():
 run(['./run.sh','init']);CREATED.append(TOPIC)
 if LANG=='go':cmd=[str(ROOT/'.cache/demo')]
 else:
  java=os.environ.get('JAVA_HOME','/opt/homebrew/opt/openjdk/libexec/openjdk.jdk/Contents/Home')+'/bin/java'
  if not pathlib.Path(java).exists():java=shutil.which('java')
  cmd=[java,'-Dorg.slf4j.simpleLogger.defaultLogLevel=error','-cp',str(ROOT/'target/classes')+':'+(ROOT/'target/classpath.txt').read_text().strip(),'course.App']
 def demo(*args,**kw):return run(cmd+list(args),**kw)
 if CHAPTER==5:
  demo('transaction','--abort','--prefix','aborted')
  demo('transaction','--prefix','committed')
 else:demo('produce','--mode','async' if CHAPTER==2 else 'sync')
 output=demo('consume','--max','6')
 records=[json.loads(x.split(' ',3)[3]) for x in output.splitlines() if x.startswith('EVENT ')]
 assert len(records)==6,output
 assert len({r['event_id'] for r in records})==6
 assert sum(r['amount_cents'] for r in records if r['type']=='OrderPaid')==3000
 assert all(r['schema_version']==1 for r in records)
 if CHAPTER==5:assert all(r['order_id'].startswith('committed') for r in records)
 # 同一分区内每个订单必须先创建再支付。
 for order in {r['order_id'] for r in records}:assert [r['order_version'] for r in records if r['order_id']==order]==[1,2]
 if CHAPTER==3:
  other=demo('consume','--group',TOKEN+'-other','--max','6');assert 'CONSUMED 6' in other
 if CHAPTER==6:
  s=demo('snapshot','--group',TOKEN+'-snapshot','--max','6');assert s.count('STATE order=')==3 and s.count('version=2')==3
  demo('tombstone');s=demo('snapshot','--group',TOKEN+'-tombstone','--max','7');assert s.count('STATE order=')==2
 if CHAPTER==7:
  s=demo('aggregate','--group',TOKEN+'-aggregate','--max','6');assert sum(int(x) for x in re.findall(r'amount_cents=(\d+)',s))==3000
 if CHAPTER==8:
  s=demo('lag');assert 'lag=unknown' not in s and all(int(x)==0 for x in re.findall(r'lag=(\d+)',s)),s
 print(f'PASS {LANG}/{ROOT.name}: events, business order, '+('transaction isolation' if CHAPTER==5 else 'chapter behavior'))

if __name__=='__main__':
 try:main()
 finally:
  cli=shutil.which('kafka-topics') or shutil.which('kafka-topics.sh')
  if cli and os.environ.get('KEEP_TEST_TOPICS')!='1':
   for topic in CREATED:subprocess.run([cli,'--bootstrap-server',BROKERS,'--delete','--topic',topic],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=30)
