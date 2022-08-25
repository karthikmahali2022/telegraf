# How to run?

   In a Ubuntu/Debian machine, install debian package as below:
   
         dpkg -i <package.deb>
         eg., dpkg -i telegraf-nd_1.24.0-a57434eb-0_amd64.deb
   
   ### Logs of influxdb:
   Install influxdb as per article https://computingforgeeks.com/how-to-install-influxdb-on-debian-linux/
   In influxdb, after "Configure InfluxDB http Authentication ",  create username as telegraf, password as 'metricsmetricsmetricsmetrics'
   
    sudo apt update
    sudo apt install -y gnupg2 curl wget
    wget -qO- https://repos.influxdata.com/influxdb.key | sudo apt-key add -
    echo "deb https://repos.influxdata.com/debian $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/influxdb.list

    sudo apt update
    sudo apt install -y influxdb

    sudo nano /etc/influxdb/influxdb.conf 
    [http]
    auth-enabled = true  
    (line no. 263)

    curl -XPOST "http://localhost:8086/query" --data-urlencode "q=CREATE USER telegraf WITH PASSWORD 'metricsmetricsmetricsmetrics' WITH ALL PRIVILEGES"
 
    Check connecting to influxdb
    influx -username 'telegraf' -password 'metricsmetricsmetricsmetrics'

    curl -G http://localhost:8086/query -u telegraf:metricsmetricsmetricsmetrics --data-urlencode "q=SHOW DATABASES"
  
    - end of influxdb config----
   
  ### Check netstat 
    
      Run command : netstat -pantul    
      You should be able to see listening ports 8086,8088

          mahalingam@telegraf-three:~$ netstat -pantul
          Proto Recv-Q Send-Q Local Address           Foreign Address         State       PID/Program name    
          tcp        0      0 0.0.0.0:22              0.0.0.0:*               LISTEN      -                   
          tcp        0      0 127.0.0.1:8088          0.0.0.0:*               LISTEN      -                   
          tcp        0      0 10.182.0.4:59958        169.254.169.254:80      ESTABLISHED -                   
          tcp        0      0 10.182.0.4:59242        142.250.141.95:443      ESTABLISHED -                   
          tcp        0    400 10.182.0.4:22           172.253.211.63:50098    ESTABLISHED -                   
          tcp        0      0 10.182.0.4:47050        216.239.34.174:443      ESTABLISHED -                   
          tcp        0      0 10.182.0.4:60048        169.254.169.254:80      ESTABLISHED -                   
          tcp6       0      0 :::8086                 :::*                    LISTEN      -                   
          tcp6       0      0 :::22                   :::*                    LISTEN      -                   
          tcp6       0      0 ::1:35572               ::1:8086                ESTABLISHED -                   
          tcp6       0      0 ::1:8086                ::1:35572               ESTABLISHED -                   
          udp        0      0 0.0.0.0:68              0.0.0.0:*                           -                   
          udp        0      0 127.0.0.1:323           0.0.0.0:*                           -                   
          udp6       0      0 ::1:323                 :::*                                -                       

### Files changed :
        deleted:    scripts/telegraf.service
        deleted:    etc/logrotate.d/telegraf
        deleted:    etc/telegraf.conf
        modified:   Makefile
        modified:   scripts/deb/post-install.sh
        modified:   scripts/deb/post-remove.sh
        modified:   scripts/deb/pre-install.sh
        modified:   scripts/deb/pre-remove.sh
        modified:   scripts/init.sh     
        
      New files: 
              etc/logrotate.d/telegraf-nd
              etc/telegraf-nd.conf
              scripts/telegraf-nd.service
       
        Changes
          [changes for package telegraf-nd](https://github.com/influxdata/telegraf/commit/190215f6a821ecb316c0d88d74ea5ad7fed3d726)                         
    
# How to create .deb package?
 Create a new VM in gcp - e2 medium, Intel(amd64) , Disk storage atleast 20GB ssd , Ubuntu/Debian11 or any other versions in same OS
       
 Install telegraf by public deb package https://github.com/influxdata/telegraf/releases
 Modify folders and filename of this installation as telegraf-nd
       
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./etc/
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./etc/telegraf-nd/
          drwxr-xr-x 0/0               0 2022-08-25 09:53 ./etc/telegraf-nd/telegraf-nd.d/
          -rw-r--r-- 0/0          381374 2022-08-25 10:40 ./etc/telegraf-nd/telegraf-nd.conf.sample
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./etc/logrotate.d/
          -rw-r--r-- 0/0             131 2022-08-25 10:40 ./etc/logrotate.d/telegraf-nd
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./opt/
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./opt/bin/
          -rwxr-xr-x 0/0       151711744 2022-08-25 10:40 ./opt/bin/telegraf-nd
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./opt/lib/
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./opt/lib/telegraf-nd/
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./opt/lib/telegraf-nd/scripts/
          -rw-r--r-- 0/0             560 2022-08-25 10:40 ./opt/lib/telegraf-nd/scripts/telegraf-nd.service
          -rwxr-xr-x 0/0            5839 2022-08-25 10:40 ./opt/lib/telegraf-nd/scripts/init.sh
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./usr/
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./usr/share/
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./usr/share/doc/
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./usr/share/doc/telegraf-nd/
          -rw-r--r-- 0/0             144 2022-08-25 10:40 ./usr/share/doc/telegraf-nd/changelog.gz
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./var/
          drwxr-xr-x 0/0               0 2022-08-25 10:40 ./var/log/
          drwxr-xr-x 0/0               0 2022-08-25 09:53 ./var/log/telegraf-nd/
       
   ### Dependencies 
       Install git, wget, curl
         Clone the repo & switch to branch telegraf-nd
         go to root folder. i.e telegraf/
       
           Install dependencies:  
                1.go 1.18/1.19 or greater
                2.sudo apt-get install --reinstall build-essential
            
   #### Run this command 
           
           make package include_packages="amd64.deb"
           Package will be in telegraf(repo)/build/dist (filename will be like telegraf*.deb)
       
           Check how to run section above, to run in a VM.
       
       
       
          

       
       
       
