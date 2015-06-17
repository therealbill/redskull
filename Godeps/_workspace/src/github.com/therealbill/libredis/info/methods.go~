package info

import (
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// BuildAllInfoMap will call `INFO ALL` and return a mapping of map[string]string for each section in the output
func BuildAllInfoMap(infostring string) map[string]map[string]string {
	lines := strings.Split(infostring, "\r\n")
	allmap := make(map[string]map[string]string)
	var sectionname string
	for _, line := range lines {
		if len(line) > 0 {
			if strings.Contains(line, "# ") {
				sectionname = strings.Split(line, "# ")[1]
				allmap[sectionname] = make(map[string]string)
			} else {
				splits := strings.Split(line, ":")
				key := splits[0]
				val := splits[1]
				secmap := allmap[sectionname]
				if secmap == nil {
					allmap[sectionname] = make(map[string]string)
				}
				allmap[sectionname][key] = val
			}
		}
	}
	return allmap
}

// BuildMapFromInfoString will take the string from a Redis info call and
// return a map[string]string
func BuildMapFromInfoString(input string) map[string]string {
	imap := make(map[string]string)
	lines := strings.Split(input, "\r\n")
	for _, line := range lines {
		if len(line) > 0 {
			if strings.Contains(line, "#") {
				imap["section"] = strings.Split(line, "#")[1]
			} else {
				splits := strings.Split(line, ":")
				key := splits[0]
				val := splits[1]
				imap[key] = val
			}
		}
	}
	return imap
}

// BuildInfoKeyspace builds out the InfoKeyspace struct
func BuildInfoKeyspace(raw string) InfoKeyspace {
	var keyspace InfoKeyspace
	lines := strings.Split(raw, "\r\n")
	section := ""
	for _, line := range lines {
		if len(line) > 0 {
			if strings.Contains(line, "#") {
				section = strings.Split(line, "# ")[1]
			} else {
				if section == "Keyspace" {
					record := strings.Split(line, ":")
					db := record[0][2:]
					data := record[1]
					splits := strings.Split(data, ",")
					imap := make(map[string]int64)
					dbIndex, _ := strconv.ParseInt(db, 10, 32)
					imap["db"] = dbIndex
					for _, entry := range splits {
						keyvals := strings.Split(entry, "=")
						key := keyvals[0]
						val, _ := strconv.ParseInt(keyvals[1], 10, 32)
						imap[key] = val
					}
					keyspace.Databases = append(keyspace.Databases, imap)
				}
			}
		} else {
			section = ""
		}
	}
	return keyspace
}

// CommandStats returns the InfoCommandStats struct
func CommandStats(commandstatstring string) (cmdstat InfoCommandStats) {
	lines := strings.Split(commandstatstring, "\r\n")
	section := ""
	for _, line := range lines {
		if len(line) > 0 {
			if strings.Contains(line, "# Commandstats") {
				cmdstat.Stats = make(map[string]map[string]float64)
				section = "Commandstats"
			} else {
				if section == "Commandstats" {
					record := strings.Split(line, ":")
					command := strings.Split(record[0], "_")[1]
					data := record[1]
					imap := make(map[string]float64)
					splits := strings.Split(data, ",")
					for _, entry := range splits {
						keyvals := strings.Split(entry, "=")
						key := keyvals[0]
						val, _ := strconv.ParseFloat(keyvals[1], 64)
						imap[key] = val
					}
					cmdstat.Stats[command] = imap
				}
			}
		} else {
			section = ""
		}
	}
	return
}

// KeyspaceStats returns an InfoKeyspace struct for accessing keyspace stats
func KeyspaceStats(infostring string) (stats InfoKeyspace) {
	trimmed := strings.TrimSpace(infostring)
	stats = BuildInfoKeyspace(trimmed)
	return
}

// GetAllInfo retuns a RedisInfoAll struct. This scturb has as it's members a
// strict for each INFO section. Since the cost difference of running INFO all
// and INFO <section> is negligible it mkes sens to just return it all and let
// the user select what info they want out of it.
func GetAllInfo(infostring string) (info RedisInfoAll) {
	allmap := BuildAllInfoMap(infostring)
	var alldata RedisInfoAll
	all := reflect.ValueOf(&alldata).Elem()
	for i := 0; i < all.NumField(); i++ {
		p := all.Type().Field(i)
		section := all.FieldByName(p.Name)
		typeOfT := section.Type()
		info := allmap[p.Name]
		s := section
		switch p.Name {
		case "Keyspace":
			alldata.Keyspace = KeyspaceStats(infostring)
		case "Commandstats":
			alldata.Commandstats = CommandStats(infostring)
		default:
			for i := 0; i < s.NumField(); i++ {
				p := typeOfT.Field(i)
				f := s.Field(i)
				tag := p.Tag.Get("redis")
				if p.Name == "Slaves" && tag == "slave*" {
					for k, v := range info {
						if strings.Contains(k, "slave") && strings.Contains(v, "ip=") {
							idstring := strings.TrimLeft(k, "slave")
							_, err := strconv.Atoi(string(idstring))
							if err == nil {
								var slave InfoSlaves
								pairs := strings.Split(v, ",")
								for _, pstring := range pairs {
									pdata := strings.Split(pstring, "=")
									switch pdata[0] {
									case "ip":
										slave.IP = pdata[1]
									case "port":
										port, _ := strconv.Atoi(pdata[1])
										slave.Port = port
									case "state":
										slave.State = pdata[1]
									case "offset":
										num, _ := strconv.Atoi(pdata[1])
										slave.Offset = num
									case "lag":
										num, _ := strconv.Atoi(pdata[1])
										slave.Lag = num
									}
								}
								alldata.Replication.Slaves = append(alldata.Replication.Slaves, slave)
							}
						}
					}
				}
				if len(info[tag]) != 0 {
					if f.Type().Name() == "int" {
						val, err := strconv.ParseInt(info[tag], 10, 64)
						if err == nil {
							f.SetInt(val)
						}
					}
					if f.Type().Name() == "float64" {
						val, err := strconv.ParseFloat(info[tag], 64)
						if err == nil {
							f.SetFloat(val)
						}
					}
					if f.Type().Name() == "string" {
						f.SetString(info[tag])
					}
					if f.Type().Name() == "bool" {
						val, err := strconv.ParseInt(info[tag], 10, 64)
						if err == nil && val != 0 {
							f.SetBool(true)
						}
					}
				}
			}
		}
	}
	return alldata

}

func upperFirst(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}
