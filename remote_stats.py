import paramiko

host = "192.168.50.94"
user = "root"
password = "ckp800810"

cmd = "\n".join([
    "echo ===uptime===",
    "uptime",
    "echo ===mem===",
    "cat /proc/meminfo | head -n 5",
    "echo ===docker_stats===",
    "docker stats --no-stream lowres-proxy --format 'table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}'",
    "echo ===loadavg===",
    "cat /proc/loadavg",
])

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(host, username=user, password=password, timeout=10)
_, stdout, stderr = client.exec_command(cmd)
print(stdout.read().decode("utf-8", errors="ignore"))
err = stderr.read().decode("utf-8", errors="ignore")
if err.strip():
    print("[stderr]")
    print(err)
client.close()
