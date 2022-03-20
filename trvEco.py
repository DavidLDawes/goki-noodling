with open('spin1.txt', 'r') as reader:
	line = reader.readline()
	while line != '':  # The EOF char is an empty string
		print(line)
		fields = line.split("\t")
		name = fields[0]
		x = fields[1]
		y = fields[2]
		starport  = fields[3]
		size  = fields[4]
		atmposphere  = fields[5]
		hydrosphere  = fields[6]
		population = fields[7]
		government = fields[8]
		law = fields[9]
		print "Name %s location (%s, %s) Starport %s Population %s" % (name, x, y, starport, population)
		line = reader.readline()

