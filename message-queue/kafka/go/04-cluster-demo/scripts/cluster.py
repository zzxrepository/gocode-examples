#!/usr/bin/env python3
"""本机三个 KRaft 节点；只操作当前 demo 的 .cache/cluster 数据和进程。"""
import os,pathlib,shutil,signal,socket,subprocess,sys
ROOT=pathlib.Path(__file__).resolve().parent.parent
STATE=ROOT/'.cache/cluster'
def executable(name):
 p=shutil.which(name) or shutil.which(name+'.sh')
 if not p:raise RuntimeError('Kafka command not found: '+name)
 return p
def owned(pid,config):
 p=subprocess.run(['ps','-p',str(pid),'-o','command='],capture_output=True,text=True)
 return p.returncode==0 and str(config) in p.stdout
def main():
 if len(sys.argv)!=2 or sys.argv[1] not in ('start','stop','status'):raise RuntimeError('usage: cluster.py start|stop|status')
 STATE.mkdir(parents=True,exist_ok=True);mode=sys.argv[1]
 if mode=='start':
  for port in (19192,19292,19392,19193,19293,19393):
   with socket.socket() as s:
    if s.connect_ex(('127.0.0.1',port))==0:raise RuntimeError(f'port {port} in use; stop the other cluster demo first')
  identifier=STATE/'cluster-id'
  if not identifier.exists():identifier.write_text(subprocess.check_output([executable('kafka-storage'),'random-uuid'],text=True).strip())
  voters=','.join(f'{i}@127.0.0.1:{19093+i*100}' for i in (1,2,3))
  for i in (1,2,3):
   node=STATE/str(i);node.mkdir(exist_ok=True);data=node/'data';conf=node/'server.properties'
   conf.write_text(f'''process.roles=broker,controller
node.id={i}
controller.quorum.voters={voters}
listeners=PLAINTEXT://127.0.0.1:{19092+i*100},CONTROLLER://127.0.0.1:{19093+i*100}
advertised.listeners=PLAINTEXT://127.0.0.1:{19092+i*100}
listener.security.protocol.map=PLAINTEXT:PLAINTEXT,CONTROLLER:PLAINTEXT
inter.broker.listener.name=PLAINTEXT
controller.listener.names=CONTROLLER
log.dirs={data}
num.partitions=3
default.replication.factor=3
min.insync.replicas=2
offsets.topic.replication.factor=3
transaction.state.log.replication.factor=3
transaction.state.log.min.isr=2
group.initial.rebalance.delay.ms=0
''')
   if not (data/'meta.properties').exists():subprocess.run([executable('kafka-storage'),'format','-t',identifier.read_text().strip(),'-c',str(conf)],check=True)
   with (node/'server.log').open('ab') as log:p=subprocess.Popen([executable('kafka-server-start'),str(conf)],stdout=log,stderr=log,env=dict(os.environ,KAFKA_HEAP_OPTS='-Xms128m -Xmx256m'),start_new_session=True)
   (node/'pid').write_text(str(p.pid))
  print('started; brokers=127.0.0.1:19192,127.0.0.1:19292,127.0.0.1:19392')
  print('wait for metadata readiness: kafka-topics --bootstrap-server 127.0.0.1:19192 --list')
 else:
  for i in (1,2,3):
   node=STATE/str(i);f=node/'pid'
   if not f.exists():continue
   pid=int(f.read_text());active=owned(pid,node/'server.properties')
   if mode=='stop' and active:os.kill(pid,signal.SIGTERM)
   print(f'node={i} pid={pid} running={active}')
if __name__=='__main__':
 try:main()
 except Exception as e:print('ERROR:',e,file=sys.stderr);sys.exit(1)
