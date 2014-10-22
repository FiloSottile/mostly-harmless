/*
 * Copyright (c) 2000-2005 Apple Computer, Inc. All rights reserved.
 *
 * @APPLE_LICENSE_HEADER_START@
 *
 * The contents of this file constitute Original Code as defined in and
 * are subject to the Apple Public Source License Version 1.1 (the
 * "License").  You may not use this file except in compliance with the
 * License.  Please obtain a copy of the License at
 * http://www.apple.com/publicsource and read it before using this file.
 *
 * This Original Code and all software distributed under the License are
 * distributed on an "AS IS" basis, WITHOUT WARRANTY OF ANY KIND, EITHER
 * EXPRESS OR IMPLIED, AND APPLE HEREBY DISCLAIMS ALL SUCH WARRANTIES,
 * INCLUDING WITHOUT LIMITATION, ANY WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE OR NON-INFRINGEMENT.  Please see the
 * License for the specific language governing rights and limitations
 * under the License.
 *
 * @APPLE_LICENSE_HEADER_END@
 */
/*
cc -o nvram nvram.c -framework CoreFoundation -framework IOKit -Wall
*/

#include <stdio.h>
#include <IOKit/IOKitLib.h>
#include <IOKit/IOKitKeys.h>
#include <CoreFoundation/CoreFoundation.h>
#include <err.h>
#include <mach/mach_error.h>

// Prototypes
unsigned int Setup(char **error);
void Teardown(unsigned int gOptionsRef);
int Get(char *name, char **value, char **error, unsigned int gOptionsRef);
int Set(char *name, char *value, int length, char **error, unsigned int gOptionsRef);
int Delete(char *name, char **error, unsigned int gOptionsRef);


unsigned int Setup(char **error)
{
  io_registry_entry_t gOptionsRef;
  kern_return_t       result;
  mach_port_t         masterPort;
  
  result = IOMasterPort(bootstrap_port, &masterPort);
  if (result != KERN_SUCCESS) {
    asprintf(error, "Error getting the IOMaster port: %s", mach_error_string(result));
    return 0;
  }
  
  gOptionsRef = IORegistryEntryFromPath(masterPort, "IODeviceTree:/options");
  if (gOptionsRef == 0) {
    asprintf(error, "nvram is not supported on this system");
    return 0;
  }

  return gOptionsRef;
}

void Teardown(unsigned int gOptionsRef)
{
  IOObjectRelease(gOptionsRef);
}


int Get(char *name, char **value, char **error, unsigned int gOptionsRef)
{
  CFStringRef   nameRef;
  CFTypeRef     valueRef;
  uint32_t      length;

  nameRef = CFStringCreateWithCString(kCFAllocatorDefault, name,
                                       kCFStringEncodingUTF8);
  if (nameRef == 0) {
    asprintf(error, "Error creating CFString for key %s", name);
    return -1;
  }
  
  valueRef = IORegistryEntryCreateCFProperty(
    gOptionsRef, nameRef, 0, 0);
  if (valueRef == 0) return -2; // kIOReturnNotFound

  length = CFDataGetLength(valueRef);
  *value = malloc(length);
  if (*value == NULL) {
    asprintf(error, "Error allocating memory");
    return -1;
  }
  CFDataGetBytes(valueRef, CFRangeMake(0, length), (unsigned char *)*value);
  
  CFRelease(nameRef);
  CFRelease(valueRef);

  return length;
}


int set(char *name, char *value, int length, char **error, unsigned int gOptionsRef)
{
  CFStringRef   nameRef;
  CFTypeRef     valueRef;
  kern_return_t result;

  nameRef = CFStringCreateWithCString(kCFAllocatorDefault, name,
                                      kCFStringEncodingUTF8);
  if (nameRef == 0) {
    asprintf(error, "Error creating CFString for key %s", name);
    return -1;
  }

  valueRef = CFDataCreateWithBytesNoCopy(kCFAllocatorDefault, (const UInt8 *)value,
                                         length, kCFAllocatorDefault);
  if (valueRef == 0) {
    asprintf(error, "Error creating CF buffer");
    return -1;
  }
  result = IORegistryEntrySetCFProperty(
    gOptionsRef, nameRef, valueRef);

  CFRelease(nameRef);

  if (result != KERN_SUCCESS) {
    asprintf(error, "Error setting variable - '%s': %s",
      name, mach_error_string(result));
    return -1;
  }

  return 0;
}

int Set(char *name, char *value, int length, char **error, unsigned int gOptionsRef)
{
  int res = set(name, value, length, error, gOptionsRef);
  if (res != 0) return res;

  // Try syncing the new data to device, best effort!
  return set(kIONVRAMSyncNowPropertyKey, name, strlen(name), error, gOptionsRef);
}


int Delete(char *name, char **error, unsigned int gOptionsRef)
{
  return set(kIONVRAMDeletePropertyKey, name, strlen(name), error, gOptionsRef);
}


// PrintOFVariables()
//
//   Print all of the firmware variables.
//
// void PrintOFVariables()
// {
//   kern_return_t          result;
//   CFMutableDictionaryRef dict;
//
//   result = IORegistryEntryCreateCFProperties(gOptionsRef, &dict, 0, 0);
//   if (result != KERN_SUCCESS) {
//     errx(1, "Error getting the firmware variables: %s", mach_error_string(result));
//   }
//
//   CFDictionaryApplyFunction(dict, &PrintOFVariable, 0);
//
//   CFRelease(dict);
// }
