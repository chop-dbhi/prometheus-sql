# -*- mode: ruby -*-
# vi: set ft=ruby :
ENV['VAGRANT_DEFAULT_PROVIDER'] = 'virtualbox'

Vagrant.configure(2) do |config|
  config.vm.provider "virtualbox" do |vb|
    vb.cpus = 1
    vb.memory = "1500"
  end
  config.ssh.insert_key = false

  ################################################################################
  # Plugin: vagrant-vbguest
  # - vagrant plugin install vagrant-vbguest
  ################################################################################
  # set auto_update to false, if you do NOT want to check the correct
  # additions version when booting this machine
  config.vbguest.auto_update = true
  # do NOT download the iso file from a webserver
  config.vbguest.no_remote = true

  config.vm.define "test" do |node|
    node.vm.box = "andrewhk/centos72-docker"
    config.vm.provision "shell", inline: "sudo yum makecache"
    config.vm.provision "shell", inline: "sudo yum -y install deltarpm"
    config.vm.provision "shell", inline: "sudo yum -y upgrade"
    config.vm.provision "shell", inline: "sudo yum clean all"    
    config.vm.provision "shell", inline: "sudo systemctl restart docker"
  end  
end