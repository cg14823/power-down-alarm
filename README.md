# Power Down Alarm

Is a super simple program that just checks how long it was off.

The motivation for this is that my building has continuous power outages
and I just wanted something that I could keep running on my RasberryPI that
will automatically record them. It currently uses SQLite to record the pulses
as well as the outages.

It works by using the RTC clock to store the a pulse every minute. Every time
it restarts it checks the last pulse and if the time difference exceeds a period
it will assume there was a power cut and store the details in a database.

I am also using the GMAIL API to send myself an email whenever the power comes
back up.

The repo also contains a simple CLI tool to test the mailer component byitself.