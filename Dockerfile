# sshd on archlinux
#
# VERSION               0.0.1
 
FROM     base/archlinux:latest
MAINTAINER 	Bill Anderson <bill.anderson@rackspace.com>

# Update the repositories
RUN pacman -Syy

# Install redis
RUN pacman -S --noconfirm redis supervisor

# Expose tcp ports
EXPOSE   26379
EXPOSE	 8000

ADD docker/sentinel.conf /etc/redis/sentinel.conf
ADD docker/supervisord.conf /etc/supervisord.conf
ADD docker/supervisord /etc/supervisor.d/
ADD html/ /usr/redskull/html/
ADD redskull /usr/redskull/
 
# Run daemon
CMD ["/usr/bin/supervisord"]
