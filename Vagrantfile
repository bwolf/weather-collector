# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure(2) do |config|
  config.vm.box = 'ffuenf/debian-8.4.0-amd64'
  config.vm.box_check_update = true

  config.vbguest.auto_update = false
  config.vbguest.no_remote = true

  config.vm.provider 'virtualbox' do |vb|
    vb.memory = "2048"
  end

  config.vm.provision 'shell', inline: <<-SHELL
    apt-get update
    apt-get -y install build-essential bison zip rpm curl ruby ruby-dev \
      git mercurial
    which fpm || gem install fpm
    apt-get -y autoremove && apt-get -y autoclean
  SHELL

  config.vm.provision 'shell', privileged: false, path: 'build.sh'
end
