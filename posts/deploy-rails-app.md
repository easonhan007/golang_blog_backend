{"title": "How to create a blog without database — part3", "created": "2022-12-15", "tags": ["ruby", "ror", "redis", "deploy", "capistrano"]}

## Background

I used to use [mina](https://github.com/mina-deploy/mina) to deploy all my rails applications, but it seems that it is not incompatible with rails 7, that makes [capistrano](https://capistranorb.com/) the best and only choice for the deployment.

## Install capistrano plugins

The following plugins are required.

```ruby
# Gemfile
group :development do
  gem "capistrano", "~> 3.17", require: false
  gem "capistrano-rails", "~> 1.6", require: false
  gem 'capistrano3-puma', require: false
  gem 'capistrano-rbenv', '~> 2.2', require: false
  gem 'capistrano-bundler', '~> 2.0', require: false
  gem 'capistrano-rake', require: false
end

```

* capistrano: the primary deployment gem that provides the fundamental deployment capabilities.
* capistrano-rails: the ruby on rails plugin that contains a wide range of automatic tasks for the rails project.
* capistrano3-puma: provides puma start script and nginx configuration file template for a typical rails project. 
* capistrano-rbenv: make sure the deploy scripts work well with rbenv.
* capistrano-bundler: the bundler plugin which is quite vital because most of the commands that are executed on the remote machine are bundler commands
* capistrano-rake: provides a nice feature that allows you to execute a rake task just like a common command 


## Generate the config files

After all the capistrano plugins installed, type the following commands to generate the config files.

```
bundle exec cap install
```
This will create some files and directories.

```
├── Capfile
├── config
│   ├── deploy
│   │   ├── production.rb
│   │   └── staging.rb
│   └── deploy.rb
└── lib
    └── capistrano
            └── tasks
```

## Capfile

Modify the capfile, import all the essential plugins.

```ruby
require "capistrano/setup"
require "capistrano/deploy"
require "capistrano/scm/git"
require "capistrano/rbenv"
require "capistrano/bundler"
require 'capistrano/rails'
require 'capistrano/puma'

install_plugin Capistrano::SCM::Git
install_plugin Capistrano::Puma
install_plugin Capistrano::Puma::Systemd
install_plugin Capistrano::Puma::Nginx 

Dir.glob("lib/capistrano/tasks/*.rake").each { |r| import r }
```


## deploy.rb

Deploy.rb is the heart of capistrano's configuration, as it is fully commented, all the config fields are easy to comprehended.

```ruby

lock "~> 3.17.1"
set :application, "mylog"
set :repo_url, "git@github.com:easonhan007/mylog.git"
# Default branch is :master
set :branch, 'main'
# Default deploy_to directory is /var/www/my_app_name
set :deploy_to, "/root/web/deploy_#{fetch(:stage)}"
# Default value for linked_dirs is []
append :linked_dirs, "log", "tmp/pids", "tmp/cache", "tmp/sockets", "tmp/webpacker", "public/system", "vendor", "storage"
set :rbenv_type, :user
set :rbenv_ruby, '3.1.1'
set :migration_role, :app
```

As my blog does not have a dababase, all the configurations are quite straightforward.

## production.rb 

Just add one line that set ssh information to production.rb file.

```ruby
server "1.2.3.4", user: "ethan", roles: %w{app db web}

```

## Configure ssh login without password and sudo without password

Here is a great reference about how to set [ssh login without passowrd](http://www.linuxproblem.org/art_9.html).

For the sake of simplicity, I will start ngnix and puma as a root user, in order to prevent type in password everytime when you deploy the project, it is a good idea to grant the user account a sudo privilege without typing the password.

```
sudo vim /etc/sudoers

// append a line 
user_name ALL=(ALL) NOPASSWD: ALL
```

## config puma and nginx 

There is an elegant approach which offered by capistrano puma plugin to configure the systemd start script promptly. 

```
cap production puma:systemd:config puma:systemd:enable
```

It provides a typical nginx configure file as well by using the following command.

```
cap production puma:nginx_config
```
Capistrano puma plugin will automaticly copy the config file to ```/etc/nginx/site-available```, you can make your own modification to adapt your needs.

Then restart the nginx server.

```
systemctl stop nginx
systemctl start nginx
```

## Complete some routine configuration 

### Generate a secert key

To keep our deployment as straightforward as possible, I will simply generate a key and copy it to production.rb. Remind you this is not recommended from the perspective of security.

```
# create a key 
rake secret
```

```
#production.rb

config.secret_key_base = 'key'
```

### Fix tailwindcss compile issue 

It is likely that you will encounter a [bug](https://github.com/tailwindlabs/tailwindcss/discussions/6738) when you are compiling the static resrouces. Here is the solution.

```ruby
#production.rb
config.assets.css_compressor = nil
```

## Try to deploy the project

```
cap production deploy
```

The above command will create all the files and soft links on the remote server, download the code, compile the static files and start the puma server.

You will probably encounter some problems, for example static files compile failure and puma failed to start. Don't panic, these are easy to fix.

### Fix static files compile error

Ssh to the remote server, and go to shared directory, type the following command.

```
mkdir vendor/javascript
```

### Fix puma starting error

This is because puma does not have the permission to write the log files, ssh to the remote machine, and grant the according access right.

```
cd shared/log
chown username:usergroup *.log
```

It is all done, now deploy again and you will find that everything goes well. 


```
cap production deploy
```

## Conclusion

* Because there are many strange bugs, you will feel frustrated when you deploying the project for the first time. Take it easy, with a little research, you will find that most of the problems will be solved. 
* Use these two commands to publish the new posts. 
  * Upload the post files. ``` bundle exec cap production deploy```
  * Parse the post files. ```bundle exec cap production invoke:rake TASK=scan:parse``` 









