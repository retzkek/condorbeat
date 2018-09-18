#!/bin/bash

$(condor_config_val MASTER) -f -t >> /var/log/condor/MasterLog 2>&1 
